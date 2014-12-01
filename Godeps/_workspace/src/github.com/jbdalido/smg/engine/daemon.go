package engine

import (
	"encoding/json"
	"fmt"
	dockerclient "github.com/vrischmann/smg/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	log "github.com/vrischmann/smg/Godeps/_workspace/src/github.com/jbdalido/logrus"
	"os"
	"path"
	"time"
)

type Daemon struct {
	Hostname      string
	DockerClient  *Docker
	EtcdClient    *Etcd
	ClusterID     string
	Machines      []string
	Listen        string
	Subscriptions []string
	PullRequest   chan string
	Error         chan error
	Reload        chan bool
	Containers    []dockerclient.APIContainers
	Stop          chan bool
}

/*
	Daemon might introduce such things as :
	- pulling
	- listing
	- :active ? and if active were an image tagged by smuggler ?
*/
func (d *Daemon) Init(clusterID string) error {

	// Assume there is always a clusterid
	// Even if it's default
	d.ClusterID = clusterID

	err := d.ConnectDocker()
	if err != nil {
		return err
	}

	err = d.ConnectEtcd()
	if err != nil {
		return err
	}

	d.Error = make(chan error, 1)
	d.PullRequest = make(chan string, 50)

	return nil
}

func (d *Daemon) ConnectDocker() error {
	// Connect to docker deamon
	err := d.DockerClient.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (d *Daemon) ConnectEtcd() error {

	// Connect to etcd machines
	d.EtcdClient = NewEtcd(d.Machines)
	err := d.EtcdClient.InitSubscription()
	if err != nil {
		log.Infof("%s", err)
	}
	// Get Hostname
	d.Hostname, err = os.Hostname()
	if err != nil {
		return err
	}
	return nil
}

func (d *Daemon) Start() error {

	go func(err chan error) {
		err <- d.EtcdWatcher()
	}(d.Error)

	//keepAlive := time.NewTicker(time.Second * 20)

	for {
		select {
		case err := <-d.Error:
			return err
		case img := <-d.PullRequest:
			log.Infof("Received Pull Request %s", img)
			err := d.PullRetries(img, 5)
			if err != nil {
				log.Infof("%s", err)
			}
		/*case <-keepAlive.C:
		err := d.KeepAlive()
		if err != nil {
			log.Infof("%s", err)
		}
		err = d.ListContainers()
		if err != nil {
			log.Infof("%s", err)
		}*/
		case <-d.Reload:
			return nil

		}

	}

}

func (d *Daemon) EtcdWatcher() error {

	if len(d.Subscriptions) == 0 {
		return fmt.Errorf("No etcd key were submitted, shutting down ...")
	}

	for _, sub := range d.Subscriptions {

		p := d.ClusterID + "/subscriptions/" + path.Clean(sub)
		go func(sub string) {
			for {
				log.Infof("[ETCD] Watching etcd at %s", sub)
				err := d.EtcdClient.Watch(p, d.PullRequest)
				if err != nil {
					log.Infof("[ETCD] Reconnection attempt for channel %s", sub)
				}
			}

		}(sub)

	}
	<-d.Stop
	return nil
}

func (d *Daemon) PullRetries(img string, retries int) error {

	for i := 0; i < retries; i++ {
		err := d.Pull(img)
		if err == nil {
			return nil
		} else {
			log.Infof("%s", err)
		}
	}
	return fmt.Errorf("Can't pull image %s", img)
}

func (d *Daemon) Pull(img string) error {

	err := d.DockerClient.Pull(img)
	if err != nil {
		return fmt.Errorf("Error pulling %s, %s", img, err)
	}
	return nil
}

func (d *Daemon) KeepAlive() error {

	// Key should be elsewhere
	key := d.ClusterID + "/machines/" + d.Hostname + "/alive"
	t := time.Now().UTC()

	// Set the raw key with the last ttl
	err := d.EtcdClient.SetRawKey(key, t.Format("20060102150405"), 30)
	if err != nil {
		return err
	}
	log.Infof("Keep alive set for %s at %s", d.Hostname, key)
	return nil

}

func (d *Daemon) ListServers() []string {
	return nil
}

func (d *Daemon) ListApps(app string, host string) error {
	return nil
}

func (d *Daemon) ListContainers() error {

	// Get running containers from the docker daemon
	conts, err := d.DockerClient.ListContainers(true)
	if err != nil {
		return nil
	}

	d.Containers = conts

	// Each Containers will be set in ETCD
	for _, cont := range d.Containers {

		// Marshall the container
		json, err := json.Marshal(cont)
		if err != nil {
			return err
		}

		// Setup the key with a 40s ttl
		// If the container dies, not my problem for now
		key := d.ClusterID + "/machines/" + d.Hostname + "/containers/" + cont.ID

		go func(k string) {
			err = d.EtcdClient.SetRawKey(k, string(json), 40)
			if err != nil {
				log.Infof("Key update failed %s", k)
			}
		}(key)
	}

	log.Infof("Docker list updated")
	return nil
}
