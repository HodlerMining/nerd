package v2client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	v2payload "github.com/nerdalize/nerd/nerd/payload/v2"
)

const (
	//AuthHeader is the name of the HTTP Authorization header.
	AuthHeader = "Authorization"

	projectsPrefix = "projects"

	tasksEndpoint   = "tasks"
	tokensEndpoint  = "tokens"
	datasetEndpoint = "datasets"
	workersEndpoint = "workers"
	queuesEndpoint  = "queues"
)

//Nerd is a client for the Nerdalize API.
type Nerd struct {
	NerdConfig
	cred string
}

//NerdConfig provides config details to create a Nerd client.
type NerdConfig struct {
	Client      Doer
	JWTProvider JWTProvider
	Base        *url.URL
	Logger      Logger
	QueueOps    QueueOps
}

// Doer executes http requests.  It is implemented by *http.Client.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// QueueOps is an interface that includes queue operations.
type QueueOps interface {
	ReceiveMessages(queueURL string, maxNoOfMessages, waitTimeSeconds int) (messages []interface{}, err error)
	UnmarshalMessage(message interface{}, v interface{}) error
	DeleteMessage(queueURL string, message interface{}) error
}

//NewNerdClient creates a new Nerd client from a config object. The http.DefaultClient
//will be used as default Doer.
func NewNerdClient(conf NerdConfig) *Nerd {
	if conf.Client == nil {
		conf.Client = http.DefaultClient
	}
	if conf.Base.Path != "" && conf.Base.Path[len(conf.Base.Path)-1] != '/' {
		conf.Base.Path = conf.Base.Path + "/"
	}
	cl := &Nerd{
		NerdConfig: conf,
		cred:       "",
	}
	return cl
}

func (c *Nerd) getCredentials() (string, error) {
	if c.JWTProvider == nil {
		return "", fmt.Errorf("No JWT provider found")
	}
	if c.cred == "" || c.JWTProvider.IsExpired() {
		cred, err := c.JWTProvider.Retrieve()
		if err != nil {
			return "", err
		}
		c.cred = cred
	}
	return c.cred, nil
}

func (c *Nerd) doRequest(method, urlPath string, input, output interface{}) (err error) {
	cred, err := c.getCredentials()
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	if input != nil {
		enc := json.NewEncoder(buf)
		err = enc.Encode(input)
		if err != nil {
			return &Error{"failed to encode the request body", err}
		}
	}

	path, err := url.Parse(urlPath)
	if err != nil {
		return &Error{"invalid url path provided", err}
	}

	resolved := c.Base.ResolveReference(path)
	req, err := http.NewRequest(method, resolved.String(), buf)
	logRequest(req, c.Logger)
	if err != nil {
		return &Error{"failed to create HTTP request", err}
	}

	req.Header.Set(AuthHeader, "Bearer "+cred)
	resp, err := c.Client.Do(req)
	if err != nil {
		return &Error{"failed to perform HTTP request", err}
	}
	logResponse(resp, c.Logger)

	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if resp.StatusCode > 399 {
		errv := &v2payload.Error{}
		err = dec.Decode(errv)
		if err != nil {
			return &Error{fmt.Sprintf("failed to decode unexpected HTTP response (%s)", resp.Status), err}
		}

		return &HTTPError{
			StatusCode: resp.StatusCode,
			Err:        errv,
		}
	}

	if output != nil {
		err = dec.Decode(output)
		if err != nil {
			return &Error{fmt.Sprintf("failed to decode successfull HTTP response (%s)", resp.Status), err}
		}
	}

	return nil
}

func createPath(projectID string, elem ...string) string {
	return path.Join(projectsPrefix, projectID, path.Join(elem...))
}

//CreateWorker creates registers this client as workable capacity
// func (c *Nerd) CreateWorker(projectID string) (output *payload.WorkerCreateOutput, err error) {
// 	output = &payload.WorkerCreateOutput{}
// 	return output, c.doRequest(http.MethodPost, createPath(projectID, workersEndpoint), nil, output)
// }
//
// // DeleteWorker removes a worker
// func (c *Nerd) DeleteWorker(projectID, workerID string) (err error) {
// 	return c.doRequest(http.MethodDelete, createPath(projectID, workersEndpoint, workerID), nil, nil)
// }
//
// //CreateSession creates a new user session.
// func (c *Nerd) CreateSession(projectID string) (output *payload.SessionCreateOutput, err error) {
// 	output = &payload.SessionCreateOutput{}
// 	return output, c.doRequest(http.MethodPost, createPath(projectID, sessionsEndpoint), nil, output)
// }
//
// //CreateDataset creates a new dataset.
// func (c *Nerd) CreateDataset(projectID string) (output *payload.DatasetCreateOutput, err error) {
// 	output = &payload.DatasetCreateOutput{}
// 	return output, c.doRequest(http.MethodPost, createPath(projectID, datasetEndpoint), nil, output)
// }
//
// //GetDataset gets a dataset by ID.
// func (c *Nerd) GetDataset(projectID, id string) (output *payload.DatasetDescribeOutput, err error) {
// 	output = &payload.DatasetDescribeOutput{}
// 	return output, c.doRequest(http.MethodGet, createPath(projectID, datasetEndpoint, id), nil, output)
// }
