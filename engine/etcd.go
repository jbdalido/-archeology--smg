package engine

import (
	"fmt"
	"github.com/jbdalido/smg/Godeps/_workspace/src/github.com/coreos/go-etcd/etcd"
	log "github.com/jbdalido/smg/Godeps/_workspace/src/github.com/jbdalido/logrus"
)

type Etcd struct {
	Client   *etcd.Client
	StopChan chan bool
}

func NewEtcd(hosts []string) *Etcd {
	return &Etcd{
		Client:   etcd.NewClient(hosts),
		StopChan: make(chan bool, 1),
	}
}

func (e *Etcd) InitSubscription() error {
	_, err := e.Client.CreateDir("subscriptions", 0)
	if err != nil {
		return err
	}
	return nil
}

func (e *Etcd) Get(key string) (string, error) {
	data, err := e.Client.Get(key, true, false)
	if err != nil {
		return "", err
	}
	if data.Node != nil {
		fmt.Printf("%s\n", data.Node)
		return data.Node.Value, nil
	}
	if data.PrevNode != nil {
		return data.PrevNode.Value, nil
	}

	return "", fmt.Errorf("No value")
}

func (e *Etcd) Watch(subscription string, c chan string) error {

	r := make(chan *etcd.Response, 999)
	// First let's make sure thoses keys are directory
	_, err := e.Client.CreateDir(subscription, 0)
	if err != nil {
		log.Infof("%s %s", subscription, err)
	}

	go func(chan *etcd.Response) error {
		for {
			select {
			case <-e.StopChan:
				return nil
			case p := <-r:
				// HERE ADD ENGINE PULL SYSTEM
				if p != nil {
					log.Infof("[PULL] Received pull request on %s for %s", subscription, p.Node.Value)
					c <- p.Node.Value
				}
			}
		}
	}(r)
	_, err = e.Client.Watch(subscription, 0, true, r, e.StopChan)
	if err != nil {
		return err
	}
	return nil
}

func (e *Etcd) SetRawKey(key, value string, ttl uint64) error {
	_, err := e.Client.Set(key, value, ttl)
	if err != nil {
		return err
	}
	return nil
}

func (e *Etcd) AppendKeyDirectory(Subscription string, image string) error {
	_, err := e.Client.CreateInOrder(Subscription, image, 0)
	if err != nil {
		return err
	}
	return nil
}
