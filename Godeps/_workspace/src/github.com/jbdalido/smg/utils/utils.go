package utils

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

/*
*   Files utils
 */

const (
	BashResolver = "\\$[{}0-9a-zA-Z_]+"
)

// Open each files in a folder
func OpenFolder(path string) ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	return files, nil

}

// Open a single file
func OpenFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed Open %s\n", path)
	}

	return file, nil
}

func OpenAndReadFile(path string) ([]byte, error) {
	file, err := OpenFile(path)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return b, nil
}

/*
*   Networking utils
 */

// Parse 127.0.0.1:90 type of host
// and return proper host, port
// TODO will not work on remote ip for now
func ParseHost(host string, port int) (string, int, error) {

	var uri string

	if port == 0 {
		port = 6666
	}

	// Check if a port is provided
	address := strings.Split(host, ":")

	if len(address) > 2 {
		// Wrong adress like n:n:n...
		return "", 0, fmt.Errorf("Wrong format for %s\n", string(host))

	} else if len(address) == 2 {

		// Good format
		uri = address[0]

		// Is port valid int
		p, err := strconv.Atoi(address[1])

		if err == nil {
			port = p
		}

	} else {
		// if no port present
		uri = host
	}

	// Try resolve valid ip or hostname with port set or default
	_, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", uri, port))
	if err != nil {
		return "", 0, fmt.Errorf("Can't resolve %s", host)
	}

	return uri, port, nil
}

// Define if a port is already in use on a machine
func PortUsable(port string) bool {
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+port)
	if err != nil {
		return false
	}

	listener, err := net.ListenTCP("tcp", addr)
	defer listener.Close()
	if err != nil {
		return false
	}
	return true

}

func EnvResolver(s string) string {
	resolved := s
	// First let's match any env variable in the string
	re := regexp.MustCompile(BashResolver)
	bvars := re.FindAllStringSubmatch(s, 20)
	// Process a getEnv against the variable
	if len(bvars) > 0 {
		for _, v := range bvars {
			exp := os.ExpandEnv(v[0])
			if exp != "" {
				resolved = strings.Replace(resolved, v[0], exp, 1)
			}
		}
	}
	return resolved
}
