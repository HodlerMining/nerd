package v1payload

import "time"

const (
	//DatasetUploadStatusCreated is the created upload status
	DatasetUploadStatusCreated = "CREATED"
	//DatasetUploadStatusUploading is the uploading upload status
	DatasetUploadStatusUploading = "UPLOADING"
	//DatasetUploadStatusSuccess is the success upload status
	DatasetUploadStatusSuccess = "SUCCESS"
)

//CreateDatasetInput is used as input to dataset creation
type CreateDatasetInput struct {
	ProjectID string `json:"project_id" valid:"required"`
}

//CreateDatasetOutput is returned from creating a dataset
type CreateDatasetOutput struct {
	DatasetSummary
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

//DescribeDatasetInput is input for dataset creation
type DescribeDatasetInput struct {
	ProjectID string `json:"project_id" valid:"required"`
	DatasetID string `json:"dataset_id" valid:"required"`
}

//DescribeDatasetOutput is output for dataset creation
type DescribeDatasetOutput struct {
	DatasetSummary
}

//ListDatasetsInput is input for dataset creation
type ListDatasetsInput struct {
	ProjectID string `json:"project_id" valid:"required"`
}

//DatasetSummary is a small version of
type DatasetSummary struct {
	ProjectID    string `json:"project_id"`
	DatasetID    string `json:"dataset_id"`
	Bucket       string `json:"bucket"`
	DatasetRoot  string `json:"dataset_root"`
	ProjectRoot  string `json:"project_root"`
	UploadExpire int64  `json:"upload_expire"`
	UploadStatus string `json:"upload_status"`
}

//ListDatasetsOutput is output for dataset creation
type ListDatasetsOutput struct {
	Datasets []*DatasetSummary
}
