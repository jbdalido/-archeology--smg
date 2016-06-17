// Provide a transition from smuggler.application
// to docker.container
package engine

import (
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	dockerclient "github.com/fsouza/go-dockerclient"
)

type Docker struct {
	Host       string `yaml:"host"`
	Cert       string `yaml:"cert"`
	Key        string `yaml:"key"`
	CA         string `yaml:"ca"`
	Mode       int
	Builder    *Builder
	Client     *dockerclient.Client
	App        *Application
	Controller *Container
	Services   []*Container
}

// Usage modes
const (
	RUN    = iota // 0
	BUILD  = iota // 1
	SHARED = false
)

func (d *Docker) Connect() error {
	// TODO :
	// - Cleanup the mess
	var (
		c   *dockerclient.Client
		err error
	)
	if d.Cert != "" && d.Key != "" {
		c, err = dockerclient.NewTLSClient(d.Host, d.Cert, d.Key, d.CA)
	} else {
		c, err = dockerclient.NewClient(d.Host)
	}
	if err != nil {
		return err
	}
	err = c.Ping()
	if err != nil {
		return err
	}

	d.Client = c
	d.Builder = NewSimpleBuilder(c)

	return nil
}

func (d *Docker) InitBuilder() {
	d.Builder = NewSimpleBuilder(d.Client)
}

// Build docker image, and according the push flag, push image on repository.
// If not empty, append the given tag to the image
func (d *Docker) Build(push bool, cleanup bool, tag string) (ImageName, error) {

	// Get the name for the image
	image := GetNameFromAppWithTag(d.App, tag, BUILD)

	// Let's build the image
	err := d.BuildDockerfile(image)
	if err != nil {
		return ImageName{}, err
	}

	if push {
		err := d.Builder.PushImage(image)
		if err != nil {
			return ImageName{}, err
		}
	}

	if cleanup {
		err := d.RemoveImage(image)
		if err != nil {
			return ImageName{}, err
		}
	}

	return image, nil
}

func (d *Docker) BuildDockerfile(name ImageName) error {
	log.Infof("--> Building image %s", name.ToString())
	err := d.Builder.MakeImage(name.Dockerfile, name, true, true)
	if err != nil {
		return err
	}
	return nil

}

func (d *Docker) BuildImage(app *Application, name ImageName, env string) error {

	if app.Name == "" {
		return fmt.Errorf("Build name can't be null")
	}

	// Todo: take that away
	log.Infof("--> Building image %s", name.ToString())

	// Let's start by setting the image from the user
	// Built image can difer from testing one
	// to go from a lighter one for example
	image := app.Image

	err := d.Builder.IssetImage(image, true, d.App.Uptodate)
	if err != nil {
		return err
	}
	d.Builder.SetFrom(image)

	// Setup the run.sh script to run smuggler style
	if env != "" {
		if _, ok := app.Commands[env]; !ok {
			return fmt.Errorf("Environment %s not found.", env)
		}

		err := d.Builder.WriteRunScript("run.sh", app.Commands[env], false)
		if err != nil {
			return err
		}
		d.Builder.Copy(". /data/")
		d.Builder.AddCmd("/data/run.sh")
	}

	// Set expose ports for smuggler.yaml
	for _, port := range app.Ports {
		// Check if the port is not a binded one
		p := strings.Split(port, ":")
		if len(p) > 1 {
			port = p[1]
		}
		d.Builder.AddPort(port)
	}

	// Setup default env variables
	for _, env := range app.Env {
		e := strings.Replace(env, "=", " ", -1)
		d.Builder.AddEnv(e)
	}

	// And write the Dockerfile
	err = d.Builder.InitDockerfile("Dockerfile")
	if err != nil {
		log.Fatalf("%s", err)
	}
	// Make the image and sent it to the api
	err = d.Builder.MakeImage("Dockerfile", name, false, app.NoCache)
	if err != nil {
		log.Fatalf("%s", err)
		return err
	}

	// TODO : take that away too
	log.Debugf("Image %s is Ready", name.ToString())

	return nil
}

func (d *Docker) SetupBaseImage(app *Application) error {

	log.Infof("Dockerfile provided for image %s", app.Image)
	// Extract context for the builder
	path, file := filepath.Split(app.ImageFile)
	if file == "" {
		return fmt.Errorf("Dockerfile is empty on path %s", app.ImageFile)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("Path %s seems malformed ?", path)
	}
	// Setup a temp builder with a new context
	tmpBuilder := NewBuilder(absPath, d.Client)
	if tmpBuilder == nil {
		return fmt.Errorf("Path (%s) does not exist", absPath)
	}
	// Get a proper ImageName
	name, err := GetNameFromStr(app.Image)
	if err != nil {
		return err
	}
	err = d.Builder.IssetImage(name.ToString(), false, d.App.Uptodate)
	// Build the image
	if err != nil {
		log.Infof("Building Image %s", name.ToString())
		err = tmpBuilder.MakeImage(file, name, true, d.App.NoCache)
		if err != nil {
			return fmt.Errorf("%s", err)
		}

	}
	return nil
}

func (d *Docker) SetupContainers() error {

	// Take away logs ?
	if d.App.Git != nil {
		log.Debugf("Git detected at branch %s", d.App.Git.Branch)
	}

	if d.App.ImageFile != "" {
		err := d.SetupBaseImage(d.App)
		if err != nil {
			return err
		}
	}

	if len(d.App.Applications) > 0 {
		for _, service := range d.App.Applications {
			if service.ImageFile != "" {
				err := d.SetupBaseImage(service)
				if err != nil {
					return err
				}
			} else {
				// Pull the image if we don't have it
				err := d.Builder.IssetImage(service.Image, true, d.App.Uptodate)
				if err != nil {
					return err
				}
			}

			service.UseDockerfile = false

			if _, ok := service.Commands[d.App.Environment]; ok {
				service.UseDockerfile = true
				log.Infof("--> Building dockerfile for service %s\n", service.Image)
			}

			// Get the right name, since we know if we're using dockerfile
			name := GetNameFromApp(service, RUN)

			if service.UseDockerfile {
				d.BuildImage(service, name, d.App.Environment)
			}

			container := &Container{
				Client:   d.Client,
				Name:     service.Name,
				Image:    name,
				Hostname: service.Hostname,
				Tags:     name.Tags,
			}

			if !container.Exists(service.ID) && !container.Exists(service.Name) {

				env := []string{}
				if service.Env != nil {
					env = service.Env
				}

				err := container.SetContainerConfig(env, nil)
				if err != nil {
					return nil
				}

				err = container.SetPorts(service.Ports)
				if err != nil {
					return err
				}

				if len(service.Volumes) > 0 {
					err = container.SetBinds(service.Volumes)
					if err != nil {
						return err
					}
				}

				err = container.CreateDockerContainer()
				if err != nil {
					return err
				}

				log.Infof("Successfully created Service %s", container.Hostname)
			} else {
				container.SetProtection(true)
				// If the container is not running we'll try to start it
				if !container.IsRunning(service.ID) && !container.IsRunning(service.Name) {
					code, err := container.Start(false)
					if err != nil {
						log.Fatalf("Can't start container %s, exit with code %d", service.Name, code)
					}
					log.Infof("Successfully restarted Service %s", container.Name)
				}

				log.Infof("Successfully attached Service %s with %s", container.Hostname, container.Name)
			}
			d.Services = append(d.Services, container)

		}
	}

	image := GetNameFromApp(d.App, RUN)

	// If we use Dockerfile that mean we need to build an image
	// from the actual directory
	if d.App.UseDockerfile {
		err := d.BuildImage(d.App, image, d.App.Environment)
		if err != nil {
			return err
		}
	} else {
		d.Builder.WriteRunScript("run.sh", d.App.Commands[d.App.Environment], true)
	}

	// Setup the base container with the image name
	container := &Container{
		Client:           d.Client,
		Name:             d.App.Name,
		Image:            image,
		Hostname:         d.App.Hostname,
		WorkingDirectory: d.App.WorkingDir,
		Tags:             image.Tags,
	}

	// If a main container is already running
	// for this app, we need to kill and
	// delete it.

	if container.Exists(d.App.Name) {
		log.Infof("--> Killing previous container %s", d.App.Name)
		// We unprotect the found container, since its name
		// has been given by us (hopefully)
		container.SetProtection(false)
		// Stop it
		err := container.Stop()
		if err != nil {
			return fmt.Errorf("Can't kill existing container: %s", err)
		}
		log.Debugf("--> Removing previous container %s", d.App.Name)
		// And remove it so we can go on
		err = container.Delete(true)
		if err != nil {
			return fmt.Errorf("Can't delete existing container: %s", err)
		}
	}
	// Let's set or reset container config
	// with the right parameters
	err := container.SetContainerConfig(d.App.Env, d.App.Commands[d.App.Environment])
	if err != nil {
		return nil
	}

	err = container.SetPorts(d.App.Ports)
	if err != nil {
		return err
	}

	// If the app is not using dockerfile
	// it's using shared folders
	if !d.App.UseDockerfile {
		// Setup shared directory
		err = container.SetBinds([]string{d.App.WorkingDir + ":/data"})
		if err != nil {
			return err
		}
	}

	// If we find volumes, we set them up also
	if len(d.App.Volumes) > 0 {
		err = container.SetBinds(d.App.Volumes)
		if err != nil {
			return err
		}
	}

	// Create the containers, just have to do links
	// and start the container
	err = container.CreateDockerContainer()
	if err != nil {
		return err
	}

	// Links
	if len(d.Services) > 0 {
		container.addLinks(d.Services)
	}

	// Container is set correctly
	d.Controller = container

	return nil
}

func (d *Docker) Configure(app *Application, mode int) error {

	if app == nil {
		return fmt.Errorf("App is null")
	}
	d.App = app
	if d.Client == nil {
		return fmt.Errorf("Docker is not connected")
	}

	d.Builder = NewBuilder(d.App.WorkingDir, d.Client)
	d.Mode = mode

	return nil

}

func (d *Docker) Stop() error {

	if d.Controller != nil {
		err := d.Controller.Stop()
		if err != nil {
			log.Errorf("Error - Cannot stop %s", d.Controller.ID)
		}
	}

	for _, service := range d.Services {
		err := service.Stop()
		if err != nil {
			log.Errorf("Error - Cannot stop %s", service.ID)
		}
	}

	return nil
}

func (d *Docker) Delete() error {
	// Delete each services
	for _, service := range d.Services {
		err := service.Delete(false)
		if err != nil {
			log.Errorf("Error - Cannot delete %s", service.ID)
		}
	}

	// Delete the container
	if d.Controller != nil {

		err := d.Controller.Delete(false)
		if err != nil {
			log.Errorf("Error - Cannot delete %s", d.Controller.ID)
		}

		// Do not remove the image if we've got
		// binded volume since the image is
		// not a custom built
		if d.App.UseDockerfile {
			err = d.RemoveImage(d.Controller.Image)
			if err != nil {
				log.Errorf("%s", err)
			}
		}
	}

	return nil
}

func (d *Docker) Start() error {

	for _, service := range d.Services {
		_, err := service.Start(false)
		if err != nil {
			return err
		}
	}
	// And we're ready to run
	log.Infof("--> Running %s ...", d.Controller.Image.ToString())

	code, err := d.Controller.Start(true)
	if err != nil {
		return err
	}
	log.Infof("--> Run Exited with code %d", code)
	if code != 0 {
		return fmt.Errorf("Run failed %d", code)
	}
	log.Infof("--> Success !")

	return nil
}

// API call

func (d *Docker) ListContainers(running bool) ([]dockerclient.APIContainers, error) {
	options := dockerclient.ListContainersOptions{
		All: true,
	}
	containers, err := d.Client.ListContainers(options)
	if err != nil {
		return nil, err
	}

	return containers, nil
}

// To keep the local registry clean we're removing images
func (d *Docker) RemoveImage(img ImageName) error {
	for _, tag := range img.Tags {
		err := d.Client.RemoveImage(img.Name + ":" + tag)
		if err != nil {
			log.Debugf("ERROR removing image %s, %s", img.Name, err)
		}
		log.Infof("Remove image %s:%s", img.Name, tag)
	}
	return nil
}

func (d *Docker) Pull(name string) error {
	imageName, err := GetNameFromStr(name)
	if err != nil {
		return err
	}

	err = d.Builder.PullImage(imageName)
	if err != nil {
		return err
	}
	return nil
}
