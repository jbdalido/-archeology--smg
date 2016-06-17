package utils

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
)

var Verbose bool
var StdPre *StdPrefixed

func InitLogger(v bool) {
	StdPre = &StdPrefixed{
		Prefix: "CONT[$]",
		buf:    bytes.NewBuffer([]byte("")),
	}
	if v {
		Verbose = true
		log.SetLevel(log.DebugLevel)
	} else {
		Verbose = false
	}
}

func IsVerbose() bool {
	return Verbose
}

type StdPrefixed struct {
	Prefix string
	buf    *bytes.Buffer
}

func (s *StdPrefixed) Write(p []byte) (n int, err error) {
	n, err = s.buf.Write(p)
	if err != nil {
		return 0, err
	}

	for {
		line, err := s.buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if log.IsTerminal() {
			fmt.Printf("%s\t%s", s.Prefix, line)
		} else {
			fmt.Print(line)
		}
	}

	return len(p), nil
}
