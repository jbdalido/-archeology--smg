package engine

import (
	"fmt"
	"regexp"
	"strings"
)

type ImageName struct {
	Registry   string
	Branch     string
	Name       string
	Tags       []string
	Dockerfile string
}

const (
	DOMAIN = ".skynet"
)

func (i *ImageName) GetAllNames() []string {
	var names []string
	s := i.Name
	if len(i.Tags) > 0 {
		for _, tag := range i.Tags {
			names = append(names, s+":"+tag)
		}
	} else {
		names = append(names, s)
	}
	return names
}

func (i *ImageName) ToHostname() (string, error) {
	if i.Name == "" {
		return "", fmt.Errorf("Image Name can't be null")
	}

	// Setup tmp name
	tmp := i.Name
	t := strings.Split(i.Name, ":")
	if len(t) == 2 {
		tmp = t[0]
	}

	// Let's check if theres not a slash around
	sp := strings.Split(tmp, "/")
	if len(sp) == 2 {
		tmp = sp[1]
	}

	// Now let's check branch
	if i.Branch != "" {
		tmp = tmp + "." + i.Branch
	}

	tmp = tmp + DOMAIN

	return tmp, nil
}

func (i *ImageName) ToString() string {
	s := i.Name
	if len(i.Tags) > 0 {
		s = s + ":" + i.Tags[0]
	}
	return s
}

func GetNameFromStr(name string) (ImageName, error) {

	imageName := ImageName{}

	t := strings.Split(name, ":")

	if len(t) == 2 {
		imageName.Name = t[0]
		imageName.Tags = append(imageName.Tags, t[1])
	} else if len(t) == 1 {
		imageName.Name = name
		imageName.Tags = append(imageName.Tags, "latest")
	} else {
		return ImageName{}, fmt.Errorf("Malformed image name %s", name)
	}
	// Let's check if theres not a slash around
	sp := strings.Split(imageName.Name, "/")
	if len(sp) == 2 {
		imageName.Registry = sp[0]
	}
	return imageName, nil

}

func GetNameFromApp(app *Application, mode int) ImageName {
	i := ImageName{
		Name:       app.Name,
		Dockerfile: "Dockerfile",
	}
	// Binded Volumes
	if !app.UseDockerfile {
		i.Name = app.Image
		t := strings.Split(app.Image, ":")
		if len(t) == 2 {
			i.Name = t[0]
			i.Tags = append(i.Tags, t[1])
		}
		return i
	}
	// Build Mode
	if mode == BUILD {
		if app.ActiveBuild.Name != "" {
			i.Name = app.ActiveBuild.Name
			n := strings.Split(i.Name, "/")
			if len(n) == 2 {
				i.Registry = n[0]
			}
		}
		if app.ActiveBuild.Dockerfile != "" {
			i.Dockerfile = app.ActiveBuild.Dockerfile
		}
	}
	// Run mode
	if mode == RUN {
		i.Tags = append(i.Tags, "stmp")
	}
	// Git dependant tags
	if app.Git != nil && mode == BUILD {
		re := regexp.MustCompile("//*")

		if app.Git.Branch != "" {
			tag := re.ReplaceAllString(app.Git.Branch, ".")
			i.Tags = append(i.Tags, tag)
		}

		if app.Git.LastCommit.ID != "" {
			i.Tags = append(i.Tags, app.Git.LastCommit.Short)
		}

		if len(app.Git.Tag) > 0 {
			i.Tags = append(i.Tags, app.Git.Tag...)
		}
	}
	// Since we're not using latest we need to set it each time,
	// this way we can use the docker pull image global
	i.Tags = append(i.Tags, "latest")
	return i
}
