package v1batch

import (
	"net/http"

	v1payload "github.com/nerdalize/nerd/nerd/client/batch/v1/payload"
)

//ClientWorkerInterface is an interface for placement of project
type ClientWorkerInterface interface {
	StartWorker(projectID, host, token, capem string) (output *v1payload.StartWorkerOutput, err error)
	StopWorker(projectID string) (output *v1payload.StopWorkerOutput, err error)
}

//StartWorker will create queue
func (c *Client) StartWorker(projectID, host, token, capem string) (output *v1payload.StartWorkerOutput, err error) {
	output = &v1payload.StartWorkerOutput{}
	input := &v1payload.StartWorkerInput{
		ProjectID: projectID,
	}

	return output, c.doRequest(http.MethodPost, createPath(projectID, workersEndpoint), input, output)
}

//StopWorker will delete queue a queue with the provided id
func (c *Client) StopWorker(projectID string) (output *v1payload.StopWorkerOutput, err error) {
	output = &v1payload.StopWorkerOutput{}
	input := &v1payload.StopWorkerInput{
		ProjectID: projectID,
	}

	return output, c.doRequest(http.MethodDelete, createPath(projectID, workersEndpoint), input, output)
}
