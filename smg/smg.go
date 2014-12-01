package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/jbdalido/smg/engine"
	"github.com/jbdalido/smg/utils"
	"os"
	"os/signal"
)

var (
	smgapp      *engine.Application
	eng         *engine.Engine
	killChannel chan os.Signal
	endChannel  chan error
)

func main() {

	cliApp := cli.App{
		Name:    "smg",
		Usage:   "Run and Build docker - https://smuggler.io",
		Version: "0.4.1",
	}

	cliApp.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "docker, d",
			Value: "",
			Usage: "Docker endpoint",
		},
		cli.StringFlag{
			Name:   "config, c",
			Value:  "~/.smg.yml",
			Usage:  "Config file to use",
			EnvVar: "SMG_CONFIG",
		},
	}

	buildFlags := []cli.Flag{
		cli.StringFlag{
			Name:  "start, s",
			Value: "smg.yml",
			Usage: "Specify a different file to use for your smg run (default: smg.yml)",
		},
		cli.BoolFlag{
			Name:  "no-cache, n",
			Usage: "Disable the use of docker cache during run and build with provided dockerfiles",
		},
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "Verbose Mode",
		},
		cli.BoolFlag{
			Name:  "push, p",
			Usage: "Push images after a successful build",
		},
		cli.BoolFlag{
			Name:  "last, l",
			Usage: "Download last image for each build",
		},
		cli.BoolFlag{
			Name:  "delete, D",
			Usage: "Delete images created after a successful build",
		},
		cli.StringSliceFlag{
			Name:  "etcd",
			Value: &cli.StringSlice{},
			Usage: "ETCD Storage http endpoint",
		},
	}

	runFlags := []cli.Flag{
		cli.StringFlag{
			Name:  "start, s",
			Value: "smg.yml",
			Usage: "Specify a different file to use for your smg run (default: smg.yml)",
		},
		cli.BoolFlag{
			Name:  "no-cache, n",
			Usage: "Disable the use of docker cache during run and build with provided dockerfiles",
		},
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "Verbose Mode",
		},
		cli.StringFlag{
			Name:  "env, e",
			Value: "default",
			Usage: "Environment (commands or dockerfiles) to use for the run",
		},
		cli.StringSliceFlag{
			Name:  "override, o",
			Value: &cli.StringSlice{},
			Usage: "Environment (commands or dockerfiles) to use for the run",
		},
		cli.BoolFlag{
			Name:  "keepalive, k",
			Usage: "Keep containers alive after a run (successful or not)",
		},
		cli.BoolFlag{
			Name:  "shared-folder, S",
			Usage: "Use a shared-folder with the main container",
		},
	}

	cliApp.HideVersion = true

	cliApp.Commands = []cli.Command{
		cli.Command{
			Name:   "run",
			Usage:  "Run containers with the proper environment",
			Flags:  runFlags,
			Action: CmdRun,
		},
		cli.Command{
			Name:   "build",
			Usage:  "Build against the active git branch of the folder and the build setup of the smg file",
			Flags:  buildFlags,
			Action: CmdBuild,
		},
	}

	cliApp.Run(os.Args)

}

func Init(c *cli.Context) error {

	utils.InitLogger(c.Bool("verbose"))
	killChannel = make(chan os.Signal, 1)
	endChannel = make(chan error)

	// Start by checking if config exist
	cfg, err := engine.NewConfig(c.GlobalString("config"), c.GlobalString("docker"))
	if err != nil {
		return fmt.Errorf("Could not start smuggler with adapter %s: %s", c.GlobalString("docker"), err)
	}

	// Start the engine with the right adapter
	eng, err = engine.New(cfg)
	if err != nil {
		return fmt.Errorf("%s: %s", c.GlobalString("docker"), err)
	}

	// Setup the application
	smgapp := &engine.Application{
		FilePath:      c.String("start"),
		Repository:    cfg.Repository,
		UseDockerfile: !c.Bool("shared-folder"),
		Uptodate:      c.Bool("last"),
		NoCache:       c.Bool("no-cache"),
		KeepAlive:     c.Bool("keepalive"),
	}

	// FIXME : setup overrides
	smgapp.SetOverrides(c.StringSlice("override"))

	// Either if it's a build or a run we need to init smuggler
	err = eng.Init(smgapp)
	if err != nil {
		return fmt.Errorf("Init failed with smuggler file, %s", err)
	}

	// catch the CTRL-C
	signal.Notify(killChannel, os.Interrupt)

	return nil
}

func CmdBuild(c *cli.Context) {
	err := Init(c)
	if err != nil {
		log.Fatalf("%s", err)
	}
	go func() {
		endChannel <- eng.Build(c.Bool("push"), c.Bool("delete"), c.StringSlice("etcd"))
	}()
	select {
	case err := <-endChannel:
		{
			if err != nil {
				log.Fatalf("%s", err)
			}
		}
	case <-killChannel:
		eng.Stop()
	}
}

func CmdRun(c *cli.Context) {
	err := Init(c)
	if err != nil {
		log.Fatalf("%s", err)
	}
	go func() {
		endChannel <- eng.Run(c.String("env"))
	}()
	select {
	case err := <-endChannel:
		{
			if err != nil {
				log.Fatalf("%s", err)
			}
		}
	case <-killChannel:
		eng.Stop()
	}
}
