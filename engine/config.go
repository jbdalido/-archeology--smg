package engine

import (
	"fmt"
	log "github.com/jbdalido/smg/Godeps/_workspace/src/github.com/jbdalido/logrus"
	"github.com/jbdalido/smg/Godeps/_workspace/src/gopkg.in/yaml.v1"
	"github.com/jbdalido/smg/utils"
	"os"
	"os/user"
	"path"
	"strings"
)

type Config struct {
	Repository string  `yaml:"repository"`
	Docker     *Docker `yaml:"docker"`
}

type Repository struct {
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
}

func NewConfig(filePath string, h string) (*Config, error) {
	c := new(Config)
	dockerCfgPath := "~/"
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	if strings.Contains(filePath, "~") {
		filePath = strings.Replace(filePath, "~", usr.HomeDir, 1)
	}
	dockerCfgPath = strings.Replace(dockerCfgPath, "~", usr.HomeDir, 1)
	datas, err := utils.OpenAndReadFile(filePath)
	if err != nil {
		c := &Config{
			Repository: "",
			Docker: &Docker{
				Host: h,
			},
		}
		// TODO : find a better way to look
		host := os.Getenv("DOCKER_HOST")
		if h == "" && host != "" {
			// Properly handle the boot2Docker host
			if host != "" {
				log.Debugf("Docker detected at %s", host)
				// Get env variables
				hostTLS := os.Getenv("DOCKER_TLS_VERIFY")
				hostCertPath := os.Getenv("DOCKER_CERT_PATH")

				// Check if they exists
				if hostTLS != "" && hostCertPath != "" {
					log.Debugf("Docker Cert Path at %s", hostCertPath)
					log.Debugf("Docker SSL Mode %s", hostTLS)
					if hostTLS == "1" {
						// Setup path for key and certificates
						key := path.Clean(hostCertPath) + "/key.pem"
						cert := path.Clean(hostCertPath) + "/cert.pem"
						ca := path.Clean(hostCertPath) + "/ca.pem"
						// Test the files
						_, err := utils.OpenFile(key)
						if err != nil {
							return nil, fmt.Errorf("Can't read your boot2docker key: %s", key)
						}
						_, err = utils.OpenFile(cert)
						if err != nil {
							return nil, fmt.Errorf("Can't read your boot2docker cert: %s", cert)
						}
						_, err = utils.OpenFile(ca)
						if err != nil {
							return nil, fmt.Errorf("Can't read your boot2docker Ca cert: %s", ca)
						}
						c.Docker.Cert = cert
						c.Docker.CA = ca
						c.Docker.Key = key
						c.Docker.Host = strings.Replace(host, "tcp", "https", 1)
					}
				}

			}
		} else {
			c.Docker.Host = "unix:///var/run/docker.sock"
		}
		return c, nil
	}
	// Parse the file to the application
	if err := yaml.Unmarshal(datas, &c); err != nil {
		return nil, fmt.Errorf("Error processing %s: %s", filePath, err)
	}
	if c.Repository != "" {
		c.Repository = strings.Replace(c.Repository, "/", "", 1)
	}
	return c, nil
}
