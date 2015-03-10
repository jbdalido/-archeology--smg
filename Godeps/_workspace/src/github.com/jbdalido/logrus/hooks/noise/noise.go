package logrus_noise

import (
	"codingit.appgratuites-network.com/ops/logrus"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	//"os"
	"bytes"
	"io/ioutil"
	"strconv"
	"time"
)

// Noise Client
type NoiseHook struct {
	URI        *url.URL
	HTTPClient *http.Client
	Options    *Options
}

type Options struct {
	Slack *SlackOptions `json:"slack,omitempty"`
	Mail  *MailOptions  `json:"mail,omitempty"`
}

type SlackOptions struct {
	Channel  string `json:"channel,omitempty"`
	Username string `json:"username,omitempty"`
}

type MailOptions struct {
	Channel  string `json:"channel,omitempty"`
	Username string `json:"username,omitempty"`
}

// Noise Message
type Message struct {
	Message   string `json:"message"`
	Timestamp int64
	Level     logrus.Level `json:"level"`
	Options   *Options     `json:"options"`
}

// NewPapertrailHook creates a hook to be added to an instance of logger.
func NewNoiseHook(host, app string, options *Options) (*NoiseHook, error) {

	if host == "" {
		return nil, fmt.Errorf("Noise Host can't be null Garry")
	}
	parsedHost, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	return &NoiseHook{
		URI:        parsedHost,
		HTTPClient: newHTTPClient(parsedHost, nil),
		Options:    options,
	}, nil

}

// Fire is called when a log event is fired.
func (h *NoiseHook) Fire(entry *logrus.Entry) error {
	message := &Message{
		Message:   entry.Message,
		Timestamp: time.Now().Unix(),
		Level:     entry.Level,
		Options:   h.Options,
	}

	msg, err := json.Marshal(message)
	if err != nil {
		logrus.Errorf("noise %s", err)
		return nil
	}

	data := &url.Values{}
	data.Add("payload", string(msg))
	_, _, err = h.do(data)
	if err != nil {
		logrus.Errorf("noise: %s", err)
		return nil
	}

	return nil
}

// do the actual prepared request in request()
func (h *NoiseHook) do(data *url.Values) ([]byte, int, error) {
	var resp *http.Response
	req, err := http.NewRequest("POST", h.URI.String(), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, -1, err
	}
	// Prepare and do the request
	req.Header.Set("User-Agent", "BenNnuts")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err = h.HTTPClient.Do(req)
	if err != nil {
		return nil, -1, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, err
	}
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("NOISE ERROR %d: %s", resp.StatusCode, body)
	}
	return body, resp.StatusCode, nil
}

// Levels returns the available logging levels.
func (h *NoiseHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
	}
}

func newHTTPClient(u *url.URL, tlsConfig *tls.Config) *http.Client {
	httpTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	return &http.Client{Transport: httpTransport}
}
