package engine

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"
)

// .dockercfg related config
// This code is own by Docker
// at : https://github.com/docker/docker/blob/487a417d9fd074d0e78876072c7d1ebfd398ea7a/registry/auth.go

const CONFIGFILE = ".dockercfg"
const INDEXSERVER = "https://index.docker.io/v1/"

type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth"`
	Email         string `json:"email"`
	ServerAddress string `json:"serveraddress,omitempty"`
}

func IndexServerAddress() string {
	return INDEXSERVER
}

func decodeAuth(authStr string) (string, string, error) {
	decLen := base64.StdEncoding.DecodedLen(len(authStr))
	decoded := make([]byte, decLen)
	authByte := []byte(authStr)
	n, err := base64.StdEncoding.Decode(decoded, authByte)
	if err != nil {
		return "", "", err
	}
	if n > decLen {
		return "", "", fmt.Errorf("Something went wrong decoding auth config")
	}
	arr := strings.SplitN(string(decoded), ":", 2)
	if len(arr) != 2 {
		return "", "", fmt.Errorf("Invalid auth configuration file")
	}
	password := strings.Trim(arr[1], "\x00")
	return arr[0], password, nil
}

// load up the auth config information and return values
// FIXME: use the internal golang config parser
func (b *Builder) LoadAuthConfig(rootPath string) error {

	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	b.Privates = make(map[string]AuthConfig)
	b.rootPath = strings.Replace(rootPath, "~", usr.HomeDir, 1)

	confFile := path.Join(b.rootPath, CONFIGFILE)
	if _, err := os.Stat(confFile); err != nil {
		return nil //missing file is not an error
	}

	c, err := ioutil.ReadFile(confFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(c, &b.Privates); err != nil {

		arr := strings.Split(string(c), "\n")
		if len(arr) < 2 {
			return fmt.Errorf("The Auth config file is empty")
		}

		authConfig := AuthConfig{}
		origAuth := strings.Split(arr[0], " = ")

		authConfig.Username, authConfig.Password, err = decodeAuth(origAuth[1])
		if err != nil {
			return err
		}

		authConfig.ServerAddress = IndexServerAddress()
		b.Privates[IndexServerAddress()] = authConfig
	} else {

		for k, authConfig := range b.Privates {
			authConfig.Username, authConfig.Password, err = decodeAuth(authConfig.Auth)
			if err != nil {
				return err
			}
			authConfig.Auth = ""
			clean := strings.Replace(k, "http://", "", 1)
			clean = strings.Replace(clean, "https://", "", 1)
			b.Privates[clean] = authConfig
			authConfig.ServerAddress = k
		}
	}

	return nil
}
