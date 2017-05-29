package v1payload

//TaskSummary is a small version of
type TaskSummary struct {
	TaskID          int64  `json:"task_id"`
	WorkloadID      string `json:"workload_id"`
	Status          string `json:"status,omitempty"`
	OutputDatasetID string `json:"output_dataset_id"`
}

//StopTaskInput is input for task creation
type StopTaskInput struct {
	ProjectID  string `json:"project_id" valid:"required"`
	WorkloadID string `json:"workload_id" valid:"required"`
	TaskID     int64  `json:"task_id" valid:"required"`
}

//StopTaskOutput is output for task creation
type StopTaskOutput struct{}

//StartTaskInput is input for task creation
type StartTaskInput struct {
	ProjectID  string `json:"project_id" valid:"required"`
	WorkloadID string `json:"workload_id" valid:"required"`

	Cmd   []string          `json:"cmd"`
	Env   map[string]string `json:"env"`
	Stdin []byte            `json:"stdin"`
}

//StartTaskOutput is output for task creation
type StartTaskOutput struct {
	TaskSummary
}

//ListTasksInput is input for task creation
type ListTasksInput struct {
	ProjectID  string `json:"project_id" valid:"required"`
	WorkloadID string `json:"workload_id" valid:"required"`
}

//ListTasksOutput is output for task creation
type ListTasksOutput struct {
	Tasks []*TaskSummary
}

//PatchTaskInput is input for task update
type PatchTaskInput struct {
	ProjectID       string `json:"project_id" valid:"required"`
	WorkloadID      string `json:"workload_id" valid:"required"`
	TaskID          int64  `json:"task_id" valid:"required"`
	OutputDatasetID string `json:"output_dataset_id"`
}

//PatchTaskOutput is output for task update
type PatchTaskOutput struct {
	TaskSummary
}

//DescribeTaskInput is input for task creation
type DescribeTaskInput struct {
	ProjectID  string `json:"project_id" valid:"required"`
	WorkloadID string `json:"workload_id" valid:"required"`
	TaskID     int64  `json:"task_id" valid:"required"`
}

//DescribeTaskOutput is output for task creation
type DescribeTaskOutput struct {
	TaskSummary
	ExecutionARN   string `json:"execution_arn"`
	NumDispatches  int64  `json:"num_dispatches"`
	Result         string `json:"result,omitempty"`
	LastErrCode    string `json:"last_err_code,omitempty"`
	LastErrMessage string `json:"last_err_message,omitempty"`
}
