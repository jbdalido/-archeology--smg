package engine

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
)

type Engine struct {
	Docker    *Docker
	ClusterID string
	Config    *Config
	App       *Application
}

func New(c *Config) (*Engine, error) {

	if c == nil {
		return nil, fmt.Errorf("Config failed")
	}

	if c.Docker == nil {
		return nil, fmt.Errorf("Invalid Docker configuration")
	}

	if c.Docker.Host == "" {
		return nil, fmt.Errorf("Docker Host can't be null")
	}

	c.Docker.Mode = 1
	c.Docker.Builder = &Builder{}

	return &Engine{
		ClusterID: "default",
		Docker:    c.Docker,
		Config:    c,
	}, nil

}

func (e *Engine) Init(app *Application) error {

	err := app.Init()
	if err != nil {
		return err
	}

	e.App = app

	return nil
}

func (e *Engine) Build(push bool, cleanup bool, tag string) error {

	if len(e.App.Builds) == 0 {
		return fmt.Errorf("No build definition matches this branch in your smuggler file")
	}

	/*if !e.App.HasDockerfile() {
		return fmt.Errorf("Building system needs a Dockerfile, file not found at %s/Dockerfile", e.App.WorkingDir)
	}*/
	// Match the build definition
	// If multiple regexp matched, we're taking the first
	env, err := e.App.InitBuild(tag)
	if err != nil {
		log.Printf("%s", err)
		return nil
	}

	if e.App.ActiveBuild.Onlyif != "" {
		log.Infof("--> Running tests (%s) before building %s", e.App.ActiveBuild.Onlyif, env)
		err := e.Run(e.App.ActiveBuild.Onlyif)
		if err != nil {
			log.Errorf("Build aborted...")
			return fmt.Errorf("%s", err)
		}

	}

	err = e.Docker.Connect()
	if err != nil {
		return fmt.Errorf("Could not connect to Docker host at %s", err)
	}

	err = e.Docker.Configure(e.App, 1)
	if err != nil {
		return err
	}

	// If push is defined is the yaml
	// no question asked we push
	if e.App.ActiveBuild.Push && !push {
		push = true
	}

	// Build and push
	_, err = e.Docker.Build(push, cleanup, tag)
	if err != nil {
		return err
	}

	return nil
}

func (e *Engine) Run(env string) error {

	err := e.Docker.Connect()
	if err != nil {
		return fmt.Errorf("Could not connect to the Docker %s", err)
	}

	err = e.App.SetEnv(env)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	err = e.Docker.Configure(e.App, 0)
	if err != nil {
		return err
	}
	// Let's setup the containers with proper configurations
	err = e.Docker.SetupContainers()
	if err != nil {
		return err
	}

	// defer the stop and delete of the container
	defer e.Stop()

	// And launch the run
	return e.Docker.Start()
}

func (e *Engine) Stop() {

	log.Info("Stopping ...")
	if !e.App.KeepAlive {
		e.Docker.Stop()
		e.Docker.Delete()
	}
}
