package v1payload

//WorkloadSummary is a smaller representation of a workload
type WorkloadSummary struct {
	ProjectID      string `json:"project_id"`
	WorkloadID     string `json:"workload_id"`
	QueueURL       string `json:"queue_url"`
	Image          string `json:"image"`
	Instances      int    `json:"instances"`
	InputDatasetID string `json:"input_dataset_id"`
	CreatedAt      int64  `json:"created_at"`
}

//ListWorkloadsInput is input for workload listing
type ListWorkloadsInput struct {
	ProjectID string `json:"project_id" valid:"required"`
}

//ListWorkloadsOutput is output for workload listing
type ListWorkloadsOutput struct {
	Workloads []*WorkloadSummary
}

//DescribeWorkloadInput is input for getting workload information
type DescribeWorkloadInput struct {
	ProjectID  string `json:"project_id" valid:"required"`
	WorkloadID string `json:"workload_id"`
}

//TaskStatus represents the status of a task
type TaskStatus string

//DescribeWorkloadOutput is output for getting workload information
type DescribeWorkloadOutput struct {
	WorkloadSummary
	TaskCount map[TaskStatus]int `json:"task_count"`
	Env       map[string]string  `json:"env"`
}

//StartWorkloadInput is input for workload creation
type CreateWorkloadInput struct {
	ProjectID      string            `json:"project_id" valid:"required"`
	Image          string            `json:"image" valid:"required"`
	Instances      int               `json:"instances" valid:"required"`
	InputDatasetID string            `json:"input_dataset_id"`
	UseCuteur      bool              `json:"use_cuteur"`
	Env            map[string]string `json:"env"`
}

//StartWorkloadOutput is output for workload creation
type CreateWorkloadOutput struct {
	WorkloadSummary
}

//StopWorkloadInput is input for workload deletion
type StopWorkloadInput struct {
	ProjectID  string `json:"project_id" valid:"required"`
	WorkloadID string `json:"workload_id" valid:"required"`
}

//StopWorkloadOutput is output for workload deletion
type StopWorkloadOutput struct{}
