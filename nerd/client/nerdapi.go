package client

import (
	"fmt"
	"net/url"
	"path"

	"github.com/dghubble/sling"
	"github.com/nerdalize/nerd/nerd/payload"
)

const (
	defaultScheme   = "https"
	defaultHost     = "platform.nerdalize.net"
	defaultBasePath = ""
	defaultVersion  = "v1"
)

type NerdAPIClient struct {
	NerdAPIConfig
}

type NerdAPIConfig struct {
	Scheme   string
	Host     string
	BasePath string
	Version  string
}

func NewNerdAPI(config NerdAPIConfig) *NerdAPIClient {
	if config.Scheme == "" {
		config.Scheme = defaultScheme
	}
	if config.Host == "" {
		config.Host = defaultHost
	}
	if config.BasePath == "" {
		config.BasePath = defaultBasePath
	}
	if config.Version == "" {
		config.Version = defaultVersion
	}
	return &NerdAPIClient{
		NerdAPIConfig: config,
	}
}

func NewNerdAPIFromURL(fullURL string, version string) (*NerdAPIClient, error) {
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse url '%v': %v", fullURL, err)
	}
	return &NerdAPIClient{
		NerdAPIConfig: NerdAPIConfig{
			Scheme:   u.Scheme,
			Host:     u.Host,
			BasePath: u.Path,
			Version:  version,
		},
	}, nil
}

func (nerdapi *NerdAPIClient) url(p string) string {
	url := &url.URL{
		Scheme: nerdapi.Scheme,
		Host:   nerdapi.Host,
		Path:   path.Join(nerdapi.BasePath, p),
		//TODO: include version
		// Path:   path.Join(nerdapi.BasePath, nerdapi.Version, p),
	}
	return url.String()
}

func (nerdapi *NerdAPIClient) CreateSession(token string) (*payload.Session, error) {
	url := nerdapi.url("/sessions/" + token)
	s := &payload.Session{}
	resp, err := sling.New().
		Post(url).
		ReceiveSuccess(s)

	if err != nil {
		return nil, fmt.Errorf("failed to send request to %v (POST): %v", url, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request '%s (POST)' returned unexpected status from API: %v", url, resp.Status)
	}
	return s, nil
}

func (nerdapi *NerdAPIClient) CreateTask(image string, dataset string, awsAccessKey string, awsSecret string, args []string) error {
	// set env variables
	args = append(args, "-e=DATASET="+dataset)
	args = append(args, "-e=AWS_ACCESS_KEY_ID="+awsAccessKey)
	args = append(args, "-e=AWS_SECRET_ACCESS_KEY="+awsSecret)

	// create payload
	p := &payload.Task{
		Image:   image,
		Dataset: dataset,
		Args:    args,
	}

	// post request
	url := nerdapi.url("/tasks")
	resp, err := sling.New().
		Post(url).
		BodyJSON(p).
		ReceiveSuccess(nil)

	if err != nil {
		return fmt.Errorf("failed to send request to %v (POST): %v", url, err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API request '%s (POST)' returned unexpected status from API: %v", url, resp.Status)
	}

	return nil
}

func (nerdapi *NerdAPIClient) PatchTaskStatus(id string, ts *payload.TaskStatus) error {
	url := nerdapi.url("/tasks/" + id)
	resp, err := sling.New().
		Patch(url).
		BodyJSON(ts).
		ReceiveSuccess(nil)

	if err != nil {
		return fmt.Errorf("failed to send request to %v (PATCH): %v", url, err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API request '%s (PATCH)' returned unexpected status from API: %v", url, resp.Status)
	}

	return nil
}

func (nerdapi *NerdAPIClient) ListTaskLogs(id string) ([]string, error) {
	url := nerdapi.url("/tasks/" + id)
	t := &payload.Task{}
	resp, err := sling.New().
		Get(url).
		ReceiveSuccess(t)

	if err != nil {
		return []string{}, fmt.Errorf("failed to send request to %v (GET): %v", url, err)
	}
	if resp.StatusCode >= 400 {
		return []string{}, fmt.Errorf("API request '%s (GET)' returned unexpected status from API: %v", url, resp.Status)
	}

	return t.LogLines, nil
}

func (nerdapi *NerdAPIClient) ListTasks() (s []payload.Task, err error) {
	url := nerdapi.url("/tasks")
	resp, err := sling.New().
		Get(url).
		ReceiveSuccess(&s)

	if err != nil {
		return []payload.Task{}, fmt.Errorf("failed to send request to %v (GET): %v", url, err)
	}
	if resp.StatusCode >= 400 {
		return []payload.Task{}, fmt.Errorf("API request '%s (GET)' returned unexpected status from API: %v", url, resp.Status)
	}
	return
}
