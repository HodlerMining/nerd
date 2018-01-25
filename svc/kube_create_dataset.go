package svc

import (
	"context"

	"github.com/nerdalize/nerd/pkg/kubevisor"

	datasetsv1 "github.com/nerdalize/nerd/crd/pkg/apis/stable.nerdalize.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//CreateDatasetInput is the input to CreateDataset
type CreateDatasetInput struct {
	Name   string `validate:"printascii"`
	Bucket string `validate:"min=1"`
	Key    string `validate:"min=1"`
}

//CreateDatasetOutput is the output to CreateDataset
type CreateDatasetOutput struct {
	Name string
}

//CreateDataset will create a dataset on kubernetes
func (k *Kube) CreateDataset(ctx context.Context, in *CreateDatasetInput) (out *CreateDatasetOutput, err error) {
	if err = k.checkInput(ctx, in); err != nil {
		return nil, err
	}

	dataset := &datasetsv1.Dataset{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: datasetsv1.DatasetSpec{
			Bucket: in.Bucket,
			Key:    in.Key,
		},
	}

	err = k.visor.CreateResource(ctx, kubevisor.ResourceTypeDatasets, dataset, in.Name)
	if err != nil {
		return nil, err
	}

	return &CreateDatasetOutput{
		Name: dataset.Name,
	}, nil
}
