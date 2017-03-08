package client

import (
	"path"

	"github.com/dghubble/sling"
	"github.com/nerdalize/nerd/nerd/client/credentials"
	"github.com/nerdalize/nerd/nerd/payload"
	"github.com/pkg/errors"
)

const (
	AuthHeader = "Authorization"

	projectsPrefix = "projects"

	tasksEndpoint    = "tasks"
	sessionsEndpoint = "tokens"
	datasetEndpoint  = "datasets"
)

//NerdAPIClient is a client for the Nerdalize API.
type NerdAPIClient struct {
	NerdAPIConfig
}

type NerdAPIConfig struct {
	Credentials *credentials.NerdAPI
	URL         string
	ProjectID   string
}

func NewNerdAPI(conf NerdAPIConfig) (*NerdAPIClient, error) {
	cl := &NerdAPIClient{
		conf,
	}
	if cl.URL == "" {
		aud, err := getAudience(conf.Credentials)
		if err != nil {
			// TODO: make it a user facing err
			return nil, errors.Wrap(err, "no valid URL was provided")
		}
		cl.URL = aud
	}
	return cl, nil
}

func getAudience(cred *credentials.NerdAPI) (string, error) {
	if cred == nil {
		return "", errors.New("credentials object was nil")
	}
	claims, err := cred.GetClaims()
	if err != nil {
		return "", errors.Wrap(err, "failed to retreive nerd claims")
	}
	if claims.Audience == "" {
		return "", errors.Errorf("nerd token '%v' does not contain audience field", claims.Audience)
	}
	return claims.Audience, nil
}

//url returns the full endpoint url appended with a given path.
func (nerdapi *NerdAPIClient) url(p string) string {
	return nerdapi.URL + "/" + path.Join(projectsPrefix, nerdapi.ProjectID, p)
}

func (nerdapi *NerdAPIClient) doRequest(s *sling.Sling, result interface{}) error {
	value, err := nerdapi.Credentials.Get()
	if err != nil {
		// TODO: Is return err ok?
		return &APIError{
			Response: nil,
			Request:  nil,
			Err:      errors.Wrap(err, "failed to get credentials"),
		}
	}
	e := &payload.Error{}
	req, err := s.Request()
	if err != nil {
		return &APIError{
			Response: nil,
			Request:  nil,
			Err:      errors.Wrap(err, "could not create request"),
		}
	}
	req.Header.Add(AuthHeader, "Bearer "+value.NerdToken)
	resp, err := s.Receive(result, e)
	if err != nil {
		return &APIError{
			Response: nil,
			Request:  req,
			Err:      errors.Wrapf(err, "unexpected behaviour when making request to %v (%v), with headers (%v)", req.URL, req.Method, req.Header),
		}
	}
	if e.Message != "" {
		return &APIError{
			Response: resp,
			Request:  req,
			Err:      e,
		}
	}
	return nil
}

//CreateSession creates a new user session.
func (nerdapi *NerdAPIClient) CreateSession() (sess *payload.SessionCreateOutput, err error) {
	sess = &payload.SessionCreateOutput{}
	url := nerdapi.url(path.Join(sessionsEndpoint))
	s := sling.New().Post(url)
	err = nerdapi.doRequest(s, sess)
	return
}

//CreateTask creates a new executable task.
func (nerdapi *NerdAPIClient) CreateTask(image string, dataset string, env map[string]string) (output *payload.TaskCreateOutput, err error) {
	output = &payload.TaskCreateOutput{}
	// create payload
	p := &payload.TaskCreateInput{
		Image:       image,
		InputID:     dataset,
		Environment: env,
	}

	// post request
	url := nerdapi.url(tasksEndpoint)
	s := sling.New().
		Post(url).
		BodyJSON(p)

	err = nerdapi.doRequest(s, output)
	return
}

//PatchTaskStatus updates the status of a task.
func (nerdapi *NerdAPIClient) PatchTaskStatus(id string, ts *payload.TaskCreateInput) error {
	ts = &payload.TaskCreateInput{}
	url := nerdapi.url(path.Join(tasksEndpoint, id))
	s := sling.New().
		Patch(url).
		BodyJSON(ts)

	return nerdapi.doRequest(s, nil)
}

//ListTasks lists all tasks.
func (nerdapi *NerdAPIClient) ListTasks() (tl *payload.TaskListOutput, err error) {
	tl = &payload.TaskListOutput{}
	url := nerdapi.url(tasksEndpoint)
	s := sling.New().Get(url)
	err = nerdapi.doRequest(s, tl)
	return
}

//CreateDataset creates a new dataset.
func (nerdapi *NerdAPIClient) CreateDataset() (d *payload.DatasetCreateOutput, err error) {
	d = &payload.DatasetCreateOutput{}
	url := nerdapi.url(datasetEndpoint)
	s := sling.New().Post(url)
	err = nerdapi.doRequest(s, d)
	return
}

//GetDataset gets a dataset by ID.
func (nerdapi *NerdAPIClient) GetDataset(id string) (d *payload.DatasetDescribeOutput, err error) {
	d = &payload.DatasetDescribeOutput{}
	url := nerdapi.url(path.Join(datasetEndpoint, id))
	s := sling.New().Get(url)
	err = nerdapi.doRequest(s, d)
	return
}
