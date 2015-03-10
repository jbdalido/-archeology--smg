package main

import (
	"codingit.appgratuites-network.com/ops/logrus"
	"codingit.appgratuites-network.com/ops/logrus/hooks/noise"
)

var log = logrus.New()

func init() {
	log.Formatter = new(logrus.TextFormatter) // default
	opts := &logrus_noise.Options{
		Slack: &logrus_noise.SlackOptions{
			Channel:  "#alarming-insights",
			Username: "testApp",
		},
	}
	hook, err := logrus_noise.NewNoiseHook("http://127.0.0.1:6699/noise/alerts", "test_app", opts)
	if err != nil {
		log.Fatalf("%s", err)
	}
	log.Hooks.Add(hook)
}

func main() {
	log.Warn("test if server down")
	log.Fatalf("not down yet")
}
