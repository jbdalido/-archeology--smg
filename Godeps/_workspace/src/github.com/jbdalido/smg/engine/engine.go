package engine

import (
	"fmt"
	log "github.com/vrischmann/smg/Godeps/_workspace/src/github.com/jbdalido/logrus"
)

type Engine struct {
	Docker     *Docker
	ClusterID  string
	Config     *Config
	App        *Application
	EtcdClient *Etcd
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

func (e *Engine) Build(push bool, cleanup bool, etcd []string) error {

	if len(e.App.Builds) == 0 {
		return fmt.Errorf("No build definition matches this branch in your smuggler file")
	}

	if !e.App.HasDockerfile() {
		return fmt.Errorf("Building system needs a Dockerfile, file not found at %s/Dockerfile", e.App.WorkingDir)
	}
	// Match the build definition
	// If multiple regexp matched, we're taking the first
	env, err := e.App.InitBuild()
	if err != nil {
		return err
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
	imageName, err := e.Docker.Build(push, cleanup)
	if err != nil {
		return err
	}
	if len(etcd) > 0 && push {

		e.EtcdClient = NewEtcd(etcd)

		subs := e.App.GetSubscriptions()
		if len(subs) > 0 {
			// Get all images name to pull request
			names := imageName.GetAllNames()

			if len(names) > 0 {
				for _, name := range names {
					err := e.PullRequest(name, subs)
					if err != nil {
						log.Errorf("%s", err)
					}
				}
			}
		}
	}

	return nil
}

func (e *Engine) PullRequest(name string, chans []string) error {
	if len(chans) < 1 {
		return fmt.Errorf("No channel to send subscribe to")
	}

	for _, c := range chans {
		log.Infof("Pull Request on channel %s for %s ,%s", c, name, e.ClusterID+"/subscriptions/"+c)
		err := e.EtcdClient.AppendKeyDirectory("/"+e.ClusterID+"/subscriptions/"+c, name)
		if err != nil {
			return fmt.Errorf("%s", err)
		}
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
