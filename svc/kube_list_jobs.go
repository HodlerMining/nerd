package svc

import "context"

//ListJobsInput is the input to ListJobs
type ListJobsInput struct{}

//ListJobsOutput is the output to ListJobs
type ListJobsOutput struct{}

//ListJobs will create a job on kubernetes
func (k *Kube) ListJobs(ctx context.Context, in *ListJobsInput) (out *ListJobsOutput, err error) {
	return out, nil
}
