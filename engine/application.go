package engine

import (
	"fmt"
	log "github.com/jbdalido/logrus"
	"github.com/jbdalido/smg/utils"
	"gopkg.in/yaml.v1"
	"math/rand"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Application struct {
	ID           string
	Name         string                   `yaml:"name"`
	Image        string                   `yaml:"image"`
	ImageFile    string                   `yaml:"image_dockerfile"`
	Services     []string                 `yaml:"services"`
	Applications map[string]*Application  `yaml:"applications"`
	Ports        []string                 `yaml:"ports"`
	Env          []string                 `yaml:"env"`
	Volumes      []string                 `yaml:"volumes"`
	Commands     map[string][]string      `yaml:"commands"`
	System       map[string]*SystemConfig `yaml:"system"`
	Builds       map[string]*Build        `yaml:"build"`
	Environments map[string]*Application  `yaml:"environments"`
	Entrypoint   string                   `yaml:"entrypoint"`
	Cmd          []string                 `yaml:"cmd"`

	Uptodate      bool
	Hostname      string
	FilePath      string
	WorkingDir    string
	Git           *utils.Git
	Environment   string
	Repository    string
	Overrides     map[string]string
	KeepAlive     bool
	UseDockerfile bool
	NoCache       bool
	ActiveBuild   *Build
}

type Build struct {
	Push   bool     `yaml:"push"`
	Deploy []string `yaml:"deploy"`
	Name   string   `yaml:"name"`
	Onlyif string   `yaml:"onlyif"`
}

type SystemConfig struct {
	Cpu int
	Ram int
}

func (a *Application) Init() error {
	// Open the smuggler.yaml file
	datas, err := utils.OpenAndReadFile(a.FilePath)
	if err != nil {
		return err
	}

	// Parse the file to the application
	if err := yaml.Unmarshal(datas, &a); err != nil {
		return fmt.Errorf("Error processing %s: %s", a.FilePath, err)
	}

	if a.Name == "" {
		return fmt.Errorf("No name for your application has been provided.")
	}

	a.WorkingDir, err = filepath.Abs(filepath.Dir(a.FilePath))
	if err != nil {
		return nil
	}

	a.Git, err = utils.NewGit(a.WorkingDir)

	return nil
}

func (a *Application) InitBuild() (string, error) {

	if a.Git != nil {
		// First let's see if we've got a clean match
		// against the branch
		if _, ok := a.Builds[a.Git.Branch]; ok {
			a.ActiveBuild = a.Builds[a.Git.Branch]
			return a.Git.Branch, nil
		}

		// If not let's go all regexp to find a match
		for i, b := range a.Builds {
			r, err := regexp.Compile(i)
			if err == nil {
				match := r.MatchString(a.Git.Branch)
				if match {
					a.ActiveBuild = b
					return i, nil
				}
			}
		}
	}

	if _, ok := a.Builds["default"]; ok {
		a.ActiveBuild = a.Builds["default"]
		return "default", nil
	}

	return "", fmt.Errorf("No build setup is matching your branch in your config file")
}

func (a *Application) BuildApplications() {
	// Handle "Basics" configurations
	if len(a.Services) > 0 && len(a.Applications) == 0 {
		a.Applications = make(map[string]*Application)
		for _, service := range a.Services {
			app := &Application{
				Name:     a.getServiceName(service, a.Name, a.KeepAlive),
				Hostname: a.getHostname(service),
				Image:    a.getImageName(service),
				ID:       a.GetOverride(service),
			}

			a.Applications[app.Hostname] = app
		}

	} else if len(a.Applications) > 0 {
		// Or complicated ones
		for n, application := range a.Applications {
			if application == nil {
				log.Warnf("Application %s is empty and has not been created.", n)
				delete(a.Applications, n)
				continue
			}
			application.Name = a.getServiceName(application.Image, a.Name, a.KeepAlive)
			application.Hostname = a.getHostname(application.Image)
			application.Image = a.getImageName(application.Image)
			application.ID = a.GetOverride(application.Hostname)
		}

	}

	a.Name = a.getServiceName(a.Image, a.Name, a.KeepAlive)
	a.Hostname = a.getHostname(a.Image)
	a.Image = a.getImageName(a.Image)

}

func (a *Application) HasDockerfile() bool {
	_, err := utils.OpenFile(a.WorkingDir + "/Dockerfile")
	if err != nil {
		return false
	}
	return true
}

func (a *Application) getServiceName(service string, appName string, keepalive bool) string {
	serviceName := a.getHostname(service)
	hostname := a.getHostname(appName)
	if serviceName == "" || hostname == "" {
		return ""
	}
	name := fmt.Sprintf("%s-%s", serviceName, hostname)
	// If keepalive is on we want to simply match services names
	if !keepalive {
		rand.Seed(time.Now().UTC().UnixNano())
		name += "-" + strconv.Itoa(rand.Intn(10000)+30000)
	}

	return name
}

func (a *Application) getImageName(name string) string {
	sp := strings.Split(name, "/")
	if len(sp) == 1 {
		if a.Repository != "" {
			name = fmt.Sprintf("%s/%s", a.Repository, name)
		}
	}
	return name
}

func (a *Application) getHostname(image string) string {
	// base name to work with can be :
	name := image

	// Handle image name : "repo/image"
	temp := strings.Split(image, "/")
	if len(temp) > 1 {
		name = temp[1]
	}

	// Handle image name : "repo/image:tag" "image:tag"
	temp = strings.Split(name, ":")
	if len(temp) > 1 {
		name = temp[0]
	}
	return name
}

func (a *Application) SetEnv(env string) error {

	if _, ok := a.Commands[env]; !ok {
		return fmt.Errorf("Environment %s not found in %s", env, a.FilePath)
	}

	a.Environment = env

	a.BuildApplications()

	return nil
}

func (a *Application) SetOverrides(overrides []string) error {
	if len(overrides) > 0 {
		a.Overrides = make(map[string]string)

		for _, o := range overrides {

			ovr := strings.Split(o, ":")
			if len(ovr) != 2 {
				return fmt.Errorf("Invalid override option for %s", o)
			}

			a.Overrides[ovr[0]] = ovr[1]
		}
	}
	return nil
}

func (a *Application) GetOverride(service string) string {
	// Check if app has override ids
	if len(a.Overrides) > 0 {

		if _, ok := a.Overrides[service]; ok {
			return a.Overrides[service]
		}
	}
	return ""
}

func (a *Application) GetSubscriptions() []string {
	if a.ActiveBuild != nil {
		return a.ActiveBuild.Deploy
	}
	return nil
}

func (a *Application) GetSystemLimits(service string) (int, int) {
	// Set some default low ressources
	var (
		cpu = 1
		ram = 512
	)
	if a.System[service] != nil {

		if a.System[service].Cpu != 0 {
			cpu = a.System[service].Cpu
		}
		if a.System[service].Ram != 0 {
			ram = a.System[service].Ram
		}
	}
	return cpu, ram
}
