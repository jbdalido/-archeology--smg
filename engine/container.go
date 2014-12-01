package engine

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/jbdalido/smg/utils"
	"io"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Container struct {
	ID               string
	Image            ImageName
	Name             string
	Tags             []string
	Hostname         string
	Code             int
	Privileged       bool
	Protection       bool
	Client           *dockerclient.Client
	HostConfig       *dockerclient.HostConfig
	ContainerConfig  *dockerclient.Config
	Docker           *dockerclient.Container
	Ports            []dockerclient.Port
	WorkingDirectory string
	Links            []*Container
}

// Inspect get the container definition
// and auto-protect the container against
// deletion, to avoid killing or removing
// already existing containers
func (c *Container) Inspect(id string) error {
	cont, err := c.Client.InspectContainer(id)
	if err != nil {
		return err
	}

	// Get intels
	c.Docker = cont
	c.Name = c.Docker.Name

	return nil
}

// Exists and IsRunning are wrapper of inspect
// to get a precise state of the container
// Exists is used to kill an existing container
func (c *Container) Exists(id string) bool {
	err := c.Inspect(id)
	if err != nil {
		return false
	}
	return true
}

// Exists and IsRunning are wrapper of inspect
// to get a precise state of the container
// IsRunning is used to attach to it
func (c *Container) IsRunning(id string) bool {

	err := c.Inspect(id)
	if err != nil {
		return false
	}

	if !c.Docker.State.Running {
		return false
	}

	return true
}

// Start is starting the container and
// redirection its log to stdout
func (c *Container) Start(out bool) (int, error) {

	if !c.IsRunning(c.Docker.ID) {

		err := c.Client.StartContainer(c.Docker.ID, c.HostConfig)
		if err != nil {
			return -1, err
		}

		if out {

			c.Logs(c.Docker.ID, utils.StdPre, utils.StdPre)
			c.Code, err = c.Client.WaitContainer(c.Docker.ID)
			if err != nil {
				return -1, nil
			}

		}
	}

	return c.Code, nil
}

// Logs is calling dockerclient logs function
// into a goroutine so we're not blocker by it
func (c *Container) Logs(id string, out, err io.Writer) {
	go func() {

		c.Client.Logs(dockerclient.LogsOptions{
			Container:    id,
			OutputStream: out,
			ErrorStream:  err,
			Follow:       true,
			Stdout:       true,
			Stderr:       true,
		})

	}()
}

// SetProtection is taking protective mesure to
// disable smuggler deletion and stopping functions
func (c *Container) SetProtection(p bool) {
	c.Protection = p
}

// Stop is stopping the container
func (c *Container) Stop() error {
	if c.Protection {
		return nil
	}
	if c.IsRunning(c.Docker.ID) {
		opts := dockerclient.KillContainerOptions{ID: c.Docker.ID}
		err := c.Client.KillContainer(opts)
		log.Debugf("Killing %s", c.Image.Name)
		if err != nil {
			log.Debugf("ERROR killing %s, %s", c.Image, err)
		}
	}
	return nil
}

// Delete is deleting the container
func (c *Container) Delete(force bool) error {
	// Keepalive means protection against suppression
	if c.Protection && !force {
		return nil
	}

	if c.Exists(c.Docker.ID) {
		// short the id for prettier logs
		id := c.Docker.ID[:8]
		opts := dockerclient.RemoveContainerOptions{ID: id}

		// Remove the container with the id
		err := c.Client.RemoveContainer(opts)
		if err != nil {
			log.Debugf("ERROR removing %s, %s", c.Image, err)
		}
	}
	return nil
}

// If system limits are defined in the smuggler conf file
// we're setting them to the container
func (c *Container) SetSystemLimits(cpu int64, ram int64) error {
	c.ContainerConfig.CpuShares = cpu
	c.ContainerConfig.Memory = ram
	return nil
}

func (c *Container) CreateDockerContainer() (err error) {

	opts := dockerclient.CreateContainerOptions{
		Name:   c.Name,
		Config: c.ContainerConfig,
	}

	c.Docker, err = c.Client.CreateContainer(opts)
	if err != nil {
		return fmt.Errorf("%s %s", c.Image, err)
	}
	return nil
}

// Set
func (c *Container) SetContainerConfig(env []string, cmd []string) error {

	c.ContainerConfig = &dockerclient.Config{
		Hostname:     c.Hostname,
		Image:        c.Image.ToString(),
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	if len(env) > 0 {
		c.ContainerConfig.Env = env
	}

	if len(cmd) > 0 {
		c.ContainerConfig.WorkingDir = "/data"
		c.ContainerConfig.Cmd = []string{"/data/run.sh"}
	}

	return nil
}

func (c *Container) SetBinds(binds []string) error {

	if c.HostConfig == nil {
		c.HostConfig = &dockerclient.HostConfig{}
	}
	// Volumes are set for both smugglers and users
	for _, bind := range binds {
		b := utils.EnvResolver(bind)
		n := strings.Split(b, ":")
		if len(n) != 2 {
			log.Warningf("Malformed Volume %s", b)
			continue
		}
		// TODO - Use another hidden directory
		if strings.HasPrefix(n[1], "/data/") {
			log.Fatalf("/data is currently user by smuggler. This directory cannot be mounted")
		}

		if runtime.GOOS == "darwin" {
			if !strings.HasPrefix(n[0], "/Users") {
				log.Warnf("Volumes will not work if outside /Users with boot2docker (%s)", n[0])
			}
		}

		log.Debugf("Binding volumes %s", b)
		c.HostConfig.Binds = append(c.HostConfig.Binds, b)
	}

	return nil

}

func (c *Container) SetPorts(ports []string) error {

	if c.HostConfig == nil {
		c.HostConfig = &dockerclient.HostConfig{}
	}

	rand.Seed(time.Now().UTC().UnixNano())

	// Set Host Port Binding
	bindPorts := map[dockerclient.Port][]dockerclient.PortBinding{}
	if len(ports) > 0 {
		for _, port := range ports {
			ps := strings.Split(string(port), ":")

			var (
				hostPort string
				p        string
			)
			// TODO:
			// Set the test against the random port, set a generate
			// function to obtain a free port
			if len(ps) == 2 {
				if utils.PortUsable(c.ParsePort(ps[0])) {
					hostPort = ps[0]
				} else {
					hostPort = strconv.Itoa(rand.Intn(10000) + 30000)
				}
				p = ps[1]
			} else {
				hostPort = strconv.Itoa(rand.Intn(10000) + 30000)
				p = port
			}

			bindPorts[c.ContainerPort(p)] = []dockerclient.PortBinding{{
				HostIp:   "0.0.0.0",
				HostPort: c.ParsePort(hostPort),
			}}
			log.Infof("Opening ports %s:%s", hostPort, p)
		}
	}

	c.HostConfig.PortBindings = bindPorts

	if c.Privileged {
		c.HostConfig.Privileged = true
	}

	return nil
}

func (c *Container) ParsePort(p string) string {
	// Let's check if a protocol is asked
	lenp := strings.Split(p, "/")
	if len(lenp) > 1 {
		return lenp[0]
	}
	return p

}

func (c *Container) ContainerPort(p string) dockerclient.Port {

	// Let's validate the protocol used
	lenp := strings.Split(p, "/")

	// A protocol is set, validation
	if len(lenp) == 2 {
		if lenp[1] == "tcp" || lenp[1] == "udp" {
			return dockerclient.Port(lenp[0] + "/" + lenp[1])
		}
		return dockerclient.Port(lenp[0] + "/tcp")
	}

	// No protocol, asign tcp
	return dockerclient.Port(p + "/tcp")
}

func (c *Container) addLinks(links []*Container) {

	var linked []string

	if len(links) > 0 {
		for _, l := range links {
			linked = append(linked, l.Name+":"+l.Hostname)
			log.Infof("Linking %s", l.Hostname)
		}
		c.HostConfig.Links = linked
	}
}
