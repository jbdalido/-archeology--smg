package logrus

import (
	"bytes"
	"fmt"
	"sort"
	"time"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 34
	grey    = 90

	prefixDebug  = "-"
	prefixInfo   = "-"
	prefixWarn   = "x"
	prefixErr    = "X"
	prefixStream = "$"
)

var (
	baseTimestamp time.Time
	isTerminal    bool
	prefix        string
)

func init() {
	baseTimestamp = time.Now()
	isTerminal = IsTerminal()
}

func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

type TextFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors   bool
	DisableColors bool
}

func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {

	var keys []string
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b := &bytes.Buffer{}

	//prefixFieldClashes(entry)

	isColored := (f.ForceColors || isTerminal) && !f.DisableColors

	if isColored {
		printColored(b, entry, keys)
	} else {
		f.appendKeyValue(b, "time", entry.Time.Format(time.RFC3339))
		f.appendKeyValue(b, "level", entry.Level.String())
		f.appendKeyValue(b, "msg", entry.Message)
		for _, key := range keys {
			f.appendKeyValue(b, key, entry.Data[key])
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func printColored(b *bytes.Buffer, entry *Entry, keys []string) {
	var (
		levelColor int
		p          string
	)
	switch entry.Level {
	case WarnLevel:
		levelColor = yellow
		p = prefixWarn
	case DebugLevel:
		levelColor = grey
		p = prefixDebug
	case ErrorLevel, FatalLevel, PanicLevel:
		levelColor = red
		p = prefixErr
	default:
		levelColor = blue
		p = prefixInfo
	}

	fmt.Fprintf(b, "\x1b[%dm[%s]\x1b[0m\t%-44s ", levelColor, p, entry.Message)

	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " \x1b[%dm%s\x1b[0m=%v", levelColor, k, v)
	}
}

func (f *TextFormatter) appendKeyValue(b *bytes.Buffer, key, value interface{}) {
	switch value.(type) {
	case string, error:
		fmt.Fprintf(b, "%v=%q ", key, value)
	default:
		fmt.Fprintf(b, "%v=%v ", key, value)
	}
}

