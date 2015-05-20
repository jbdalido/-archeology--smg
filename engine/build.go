package engine

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"

	log "github.com/jbdalido/smg/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/jbdalido/smg/Godeps/_workspace/src/github.com/docker/docker/pkg/archive"
	dockerclient "github.com/jbdalido/smg/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"github.com/jbdalido/smg/utils"
)

//
type Builder struct {
	Client   *dockerclient.Client
	AppPath  string
	Path     string
	File     *Dockerfile
	rootPath string
	Hostname string
	Privates map[string]AuthConfig
}

// Dockerfile represent an actual Dockerfile to write
type Dockerfile struct {
	From       string
	Workdir    string
	Env        []string
	Run        []string
	Entrypoint string
	Ports      []string
	Add        []string
	Copy       []string
	Cmd        []string
}

func NewBuilder(p string, client *dockerclient.Client) *Builder {

	// build a temporary folder under .build
	path := path.Clean(p)

	// Check if the builder is setup against a valid folder
	_, err := utils.OpenFolder(p)
	if err != nil {
		return nil
	}

	tmpPath := "/tmp/smg/build" + strconv.Itoa(rand.Intn(10000)+30000)

	// Coy with tar stream
	err = archive.CopyWithTar(path, tmpPath)
	if err != nil {
		log.Fatalf("%s", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "undefined"
	}

	b := &Builder{
		AppPath:  path,
		Path:     tmpPath,
		Client:   client,
		File:     &Dockerfile{},
		Hostname: hostname,
	}

	// Try to charge /root/.dockercfg
	// TODO :
	// - Adapt to dynamic file location ?
	err = b.LoadAuthConfig("~/")
	if err != nil {
		log.Infof(" Unreadable config file at ~/.dockercfg")
	}

	return b
}

func NewSimpleBuilder(client *dockerclient.Client) *Builder {

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "undefined"
	}

	b := &Builder{
		Client:   client,
		File:     &Dockerfile{},
		Hostname: hostname,
	}

	// Try to charge /root/.dockercfg
	// TODO :
	// - Adapt to dynamic file location
	err = b.LoadAuthConfig("~/")
	if err != nil {
		log.Debugf("Unreadable config file at ~/.dockercfg")
	}

	return b
}

// TODO :
// 	-	split this function into two, one to Write the Dockerfile,
//		and one to read and save as []bytes any given file and then
//		use it as dockerfile. SetCommands, WriteDockerfile, ReadDockerfile
//	-	Add a method to validate it's looks like a dockerfile ?

func (b *Builder) InitDockerfile(filename string) error {
	var commands bytes.Buffer

	if b.File.From == "" {
		return fmt.Errorf("No from image")
	}

	commands.WriteString(fmt.Sprintf("FROM %s\nMaintainer Han Solo <solo@smuggler.io>\n\n",
		b.File.From))

	for _, env := range b.File.Env {
		commands.WriteString(fmt.Sprintf("ENV %s\n", env))
	}

	for _, port := range b.File.Ports {
		commands.WriteString(fmt.Sprintf("EXPOSE %s\n", port))
	}

	for _, add := range b.File.Add {
		commands.WriteString(fmt.Sprintf("ADD %s\n", add))
	}

	for _, cp := range b.File.Copy {
		commands.WriteString(fmt.Sprintf("COPY %s\n", cp))
	}

	if b.File.Workdir != "" {
		commands.WriteString(fmt.Sprintf("WORKDIR %s\n", b.File.Workdir))
	}

	for _, run := range b.File.Run {
		commands.WriteString(fmt.Sprintf("RUN %s\n", run))
	}

	if b.File.Entrypoint != "" {
		commands.WriteString(fmt.Sprintf("ENTRYPOINT [\"%s\"]\n", b.File.Entrypoint))
	}

	if len(b.File.Cmd) > 0 {
		commands.WriteString("CMD [ ")
		for _, cmd := range b.File.Cmd {
			commands.WriteString(fmt.Sprintf("\"%s\" ", cmd))
		}
		commands.WriteString("]\n")
	}

	err := b.WriteFile(fmt.Sprintf("%s/%s", b.Path, filename), commands.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (b *Builder) ReadDockerfile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed Open %s\n", path)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Writefile is writing the byte data on the system
func (b *Builder) WriteFile(path string, data []byte) error {
	err := ioutil.WriteFile(path, data, 0755)
	if err != nil {
		return err
	}
	return nil
}

func (b *Builder) WriteRunScript(name string, lines []string, sharedDirectory bool) error {
	// Write a run.sh script
	if len(lines) > 0 {
		// Setup run script line with friendly docker script
		var commands bytes.Buffer

		commands.WriteString("#!/bin/bash\n")
		// Exit immediately if a command exits with a non-zero status.
		commands.WriteString("set -e\n")
		for _, line := range lines {
			commands.WriteString(fmt.Sprintf("%s\n", line))
		}

		// Setup the right path to write the run script
		p := b.Path
		if sharedDirectory {
			p = b.AppPath
		}

		// Write the run script
		err := b.WriteFile(fmt.Sprintf("%s/%s", p, name), commands.Bytes())
		if err != nil {
			return err
		}

	}
	return nil
}

/*
*   Build commands dockerfile
 */

func (b *Builder) SetFrom(from string) error {
	b.File.From = from
	return nil
}

func (b *Builder) SetWorkdir(path string) error {
	b.File.Workdir = path
	return nil
}

func (b *Builder) AddRun(run string) error {
	b.File.Run = append(b.File.Run, run)
	return nil
}

func (b *Builder) AddEnv(env string) error {
	b.File.Env = append(b.File.Env, env)
	return nil
}

func (b *Builder) AddPort(port string) error {
	b.File.Ports = append(b.File.Ports, port)
	return nil
}

func (b *Builder) AddCmd(cmd string) error {
	b.File.Cmd = append(b.File.Cmd, cmd)
	return nil
}

func (b *Builder) Add(add string) error {
	b.File.Add = append(b.File.Add, add)
	return nil
}

func (b *Builder) Copy(cp string) error {
	b.File.Copy = append(b.File.Copy, cp)
	return nil
}

func (b *Builder) SearchFrom(path string) (ImageName, error) {
	from, err := utils.OpenFileAndRegexp(path, "^(FROM (.*))$")
	if err != nil {
		return ImageName{}, fmt.Errorf("From not found %s", err)
	}
	if len(from) == 0 {
		return ImageName{}, fmt.Errorf("From not found")
	}
	if len(from) != 3 {
		return ImageName{}, fmt.Errorf("Can't find from")
	}
	image, err := GetNameFromStr(from[2])
	if err != nil {
		return ImageName{}, err
	}
	log.Infof("From found %s", image.Name)
	return image, nil

}

func (b *Builder) MakeImage(dockerfile string, name ImageName, uptodate bool, nocache bool) error {

	// Prevent cleanup of directories
	defer b.Cleanup()

	_, err := utils.OpenAndReadFile(b.Path + "/" + dockerfile)
	if err != nil {
		return fmt.Errorf("%s does not exist", dockerfile)
	}

	// Tar the current path since
	// the Dockerfile is here
	tarDir, err := archive.Tar(b.Path, 0)
	if err != nil {
		return err
	}

	if uptodate {
		log.Infof("Search from in %s", b.Path+"/"+dockerfile)
		image, err := b.SearchFrom(b.Path + "/" + dockerfile)
		if err != nil {
			return err
		}
		err = b.PullImage(image)
		if err != nil {
			return err
		}
	}

	opts := dockerclient.BuildImageOptions{
		Name:        name.Name,
		InputStream: tarDir,
		NoCache:     nocache,
		Dockerfile:  dockerfile,
	}

	if utils.IsVerbose() {
		opts.OutputStream = utils.StdPre
	} else {
		opts.OutputStream = bytes.NewBuffer(nil)
	}
	// Send to the api
	if err := b.Client.BuildImage(opts); err != nil {
		return err
	}
	if len(name.Tags) > 0 {
		for _, tag := range name.Tags {
			// Tag Image
			opts := dockerclient.TagImageOptions{
				Tag:   tag,
				Repo:  name.Name,
				Force: true,
			}
			if err := b.Client.TagImage(name.Name, opts); err != nil {
				return err
			}
			log.Debugf("Image %s tagged %s", name.Name, tag)
		}
	}

	return nil

}

func (b *Builder) PushImage(name ImageName) error {

	auth := dockerclient.AuthConfiguration{}
	if _, ok := b.Privates[name.Registry]; ok {
		auth = dockerclient.AuthConfiguration{
			Username: b.Privates[name.Registry].Username,
			Password: b.Privates[name.Registry].Password,
		}
	}

	// Push all the tags if they exist
	if len(name.Tags) > 0 {
		for _, tag := range name.Tags {

			// Setup push options for docker client
			pushOptions := dockerclient.PushImageOptions{
				Name: name.Name,
				Tag:  tag,
			}

			// Let's push
			log.Infof("Pushing %s:%s", name.Name, tag)
			err := b.Client.PushImage(pushOptions, auth)
			if err != nil {
				return err
			}
			log.Infof("-->  Push succeed %s", name.Name)
		}
	}

	return nil
}

func (b *Builder) PullImage(name ImageName) error {
	if name.Name == "" {
		return fmt.Errorf("Name empty")
	}

	if b.Client == nil {
		return fmt.Errorf("Client lost connection")
	}

	auth := dockerclient.AuthConfiguration{}
	if _, ok := b.Privates[name.Registry]; ok {
		// We need to auth you
		auth = dockerclient.AuthConfiguration{
			Username: b.Privates[name.Registry].Username,
			Password: b.Privates[name.Registry].Password,
		}
	}

	buf := bytes.NewBuffer(nil)

	for _, tag := range name.Tags {

		p := dockerclient.PullImageOptions{
			OutputStream: buf,
			Repository:   name.Name,
			Tag:          tag,
		}
		log.Infof("Pulling image %s:%s", p.Repository, tag)
		err := b.Client.PullImage(p, auth)
		if err != nil {
			log.Infof("Error pulling image %s : %s", p.Repository, err)
			return err
		}
		log.Infof("Pull succeed %s:%s", p.Repository, tag)
	}
	return nil
}

func (b *Builder) IssetImage(image string, force bool, upToDate bool) error {
	if image == "" {
		return fmt.Errorf("Image can't be null")
	}

	// Inspect image provide a way to know it
	// the image exist
	_, err := b.Client.InspectImage(image)
	if err != nil {
		// Pull the image if there is an error
		if !force {
			return err
		}
		upToDate = true
	}

	if upToDate {
		imageName, err := GetNameFromStr(image)
		if err != nil {
			return err
		}
		err = b.PullImage(imageName)
		if err != nil {
			return err
		}
	}
	return nil

}

func (b *Builder) Cleanup() {
	if strings.HasPrefix(b.Path, "/tmp/smg") {
		log.Debugf("Cleaning up: %s", b.Path)
		os.RemoveAll(b.Path)
	}
}
