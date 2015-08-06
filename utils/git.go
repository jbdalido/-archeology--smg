package utils

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
)

type Git struct {
	Path       string
	LastCommit *Commit
	Repository string
	Branch     string
	Tag        []string
}

type Commit struct {
	User  string
	ID    string
	Short string
}

func NewGit(p string) (*Git, error) {

	g := &Git{
		LastCommit: &Commit{},
	}

	head := path.Clean(p) + "/.git/HEAD"
	s, err := OpenFileAndRegexp(head, "^(ref: refs/heads/([^ ]+).*)|(.*)$")
	if err != nil {
		return nil, err
	}

	if len(s) == 4 {
		g.Branch = s[2]
	}

	if g.Branch != "" {
		commit := path.Clean(p) + "/.git/refs/heads/" + g.Branch
		t, err := OpenFileAndRegexp(commit, "(.*)")
		if err != nil {
			return g, nil
		}
		g.LastCommit.ID = t[0]
		g.LastCommit.Short = t[0][:9]
	} else {
		g.LastCommit.ID = s[0]
		g.LastCommit.Short = s[0][:9]
	}

	tags, err := ioutil.ReadDir(path.Clean(p) + "/.git/refs/tags/")
	if err != nil {
		return g, nil
	}
	if len(tags) > 0 {
		for _, tag := range tags {
			b, _ := OpenAndReadFile(path.Clean(p) + "/.git/refs/tags/" + tag.Name())
			if string(b)[:9] == g.LastCommit.Short {
				g.Tag = append(g.Tag, tag.Name())
			}
		}
	}

	return g, nil

}

func OpenFileAndRegexp(path, rex string) ([]string, error) {

	if _, err := os.Stat(path); err == nil {

		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("Does not seems to be a git repository (bare not handled for now)")
		}
		file := bufio.NewReader(f)
		scanner := bufio.NewScanner(file)

		if scanner.Scan() {
			// Open the .git in the current folder
			text := scanner.Text()
			r := regexp.MustCompile(rex)
			s := r.FindStringSubmatch(text)

			if len(s) > 0 {
				return s, nil
			}
			return nil, fmt.Errorf("Cant match regexp")

		}
		return nil, fmt.Errorf("File empty")

	}
	return nil, fmt.Errorf("File not found")
}
