package main

import (
	"fmt"
	flag "github.com/vrischmann/smg/Godeps/_workspace/src/github.com/docker/docker/pkg/mflag"
	"github.com/vrischmann/smg/Godeps/_workspace/src/github.com/jbdalido/smg/engine"
	"log"
)

const (
	Version = "2.0.2"
)

// Support for multiple values command line options

type stringslice []string

func (s *stringslice) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *stringslice) Set(value string) error {
	if value != "" {
		*s = append(*s, value)
	}

	return nil
}

func main() {

	// setup command line arguments
	var (
		machines      stringslice
		subscriptions stringslice
		configPath    string
		dockerHost    string
		cluster       string
	)

	// Override flag with docker mflag
	flag.Var(&machines, []string{"e", "-etcd"}, "Etcd machines ex : --etcd http://10.0.0.1:5001 --etcd http://10.0.0.2:5001 ")
	flag.Var(&subscriptions, []string{"s", "-subscribe"}, "Channels to subscribe for pull requests")
	flag.StringVar(&cluster, []string{"C", "-cluster"}, "default", "Cluster ID")
	flag.StringVar(&configPath, []string{"c", "-config"}, "~/.smg.yaml", "Config file to use, if not we write ~/.smuggler.yaml")
	flag.StringVar(&dockerHost, []string{"d", "-docker"}, "", "Docker socket/host")

	flag.Parse()

	// Start by checking if config exist
	cfg, err := engine.NewConfig(configPath, dockerHost)
	if err != nil {
		log.Fatalf("Invalid config %s", err)
	}

	if cfg.Docker == nil {
		log.Fatalf("Invalid configuration")
	}
	if cfg.Docker.Host == "" {
		log.Fatalf("Docker Host can't be null")
	}
	cfg.Docker.Mode = 1
	cfg.Docker.Builder = &engine.Builder{}

	d := &engine.Daemon{
		DockerClient:  cfg.Docker,
		Machines:      machines,
		Subscriptions: subscriptions,
	}
	log.Printf("Starting smgd[%s] cluster[%s] \n", Version, cluster)
	err = d.Init(cluster)
	if err != nil {
		log.Fatalf("Err %s", err)
	}

	err = d.Start()
	if err != nil {
		log.Fatalf("Err %s", err)
	}

}

func showVersion() {
	log.Printf("Smuggler v2.3 RC1")
}
