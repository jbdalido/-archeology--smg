package engine

import (
	"log"
	"testing"
)

func TestHostname(t *testing.T) {
	i := &ImageName{
		Name:   "jbaptiste/smuggler",
		Branch: "featureapi",
		Tags:   []string{"taga"},
	}
	hostname, err := i.ToHostname()
	if err != nil {
		t.Errorf("%s", err)
	}

	i = &ImageName{
		Name:   "image",
		Branch: "tag",
		Tags:   []string{"taga"},
	}
	hostname, err = i.ToHostname()
	if err != nil {
		t.Errorf("%s", err)
	}

	i = &ImageName{
		Name: "image",
	}
	hostname, err = i.ToHostname()
	if err != nil {
		t.Errorf("%s", err)
	}
	log.Printf("Hostname is %s", hostname)

}
