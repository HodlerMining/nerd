package svc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nerdalize/nerd/pkg/kubevisor"
	"github.com/pkg/errors"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//CreateSecretInput is the input to CreateSecret
type CreateSecretInput struct {
	Image    string `validate:"printascii"`
	Registry string `validate:"required"`
	Project  string
	Tag      string
	Username string `validate:"required"`
	Password string `validate:"required"`
}

//CreateSecretOutput is the output to CreateSecret
type CreateSecretOutput struct {
	Name string
}

//CreateSecret will create a secret on kubernetes
func (k *Kube) CreateSecret(ctx context.Context, in *CreateSecretInput) (out *CreateSecretOutput, err error) {
	if err = k.checkInput(ctx, in); err != nil {
		return nil, err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"image": in.Image, "project": in.Project, "registry": in.Registry, "tag": in.Tag},
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{},
	}

	secret.Data[v1.DockerConfigJsonKey], err = transformCredentials(in.Username, in.Password, in.Registry)
	if err != nil {
		return nil, err
	}

	err = k.visor.CreateResource(ctx, kubevisor.ResourceTypeSecrets, secret, "")
	if err != nil {
		return nil, err
	}

	return &CreateSecretOutput{
		Name: secret.Name,
	}, nil
}

func transformCredentials(username, password, registry string) (dockereCfg []byte, err error) {
	var dockerCfg []byte
	auths := map[string]interface{}{}
	cfg := map[string]interface{}{
		"auths": auths,
		"HttpHeaders": map[string]interface{}{
			"User-Agent": "Docker-Client/1.11.2 (linux)",
		},
	}
	authStr := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	auths[fmt.Sprintf("https://%s/v1/", registry)] = map[string]string{
		"auth": authStr,
	}
	auths[fmt.Sprintf("%s", registry)] = map[string]string{
		"auth": authStr,
	}
	if dockerCfg, err = json.Marshal(cfg); err != nil {
		return dockerCfg, errors.Wrap(err, "failed to serialize docker secret cfg")
	}
	return dockerCfg, nil
}

// ExtractRegistry takes a string as input and divides it in image, project, registry, tag
func ExtractRegistry(image string) (string, string, string, string) {
	// Supported registries:
	// - project/image -> index.docker.io
	// - ACCOUNT.dkr.ecr.REGION.amazonaws.com/image -> aws
	// - azurecr.io/image -> azure
	// - quay.io/project/image -> quay.io
	// - gcr.io/project/image -> gcr
	// gitlab?? other providers?

	var tag string
	parts := strings.Split(image, ":")
	if len(parts) > 1 {
		tag = parts[1]
	}
	parts = strings.Split(parts[0], "/")
	switch len(parts) {
	case 2:
		if !strings.Contains(parts[0], ".") {
			return parts[1], parts[0], "index.docker.io", tag
		}
		return parts[1], "", parts[0], tag
	case 3:
		return parts[2], parts[1], parts[0], tag
	}
	return "", "", "", tag
}
