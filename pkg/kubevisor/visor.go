package kubevisor

import (
	"context"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	kuberr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

//Logger describes the logging dependency the services require
type Logger interface {
	Debugf(format string, args ...interface{})
}

//KubeResourceType is a type of Kubernetes resource
type KubeResourceType string

var (
	//KubeResourceTypeJobs is used for job management
	KubeResourceTypeJobs = KubeResourceType("jobs")
)

//KubeManagedNames allows for Nerd to transparently manage resources based on names and there prefixes
type KubeManagedNames interface {
	GetName() string
	GetLabels() map[string]string
	SetLabels(map[string]string)
	SetName(name string)
	SetGenerateName(name string)
}

//KubeListTranformer must be implemented to allow Nerd to transparently manage resource names
type KubeListTranformer interface {
	Transform(fn func(in KubeManagedNames) (out KubeManagedNames))
}

//Visor provides access to Kubernetes resources while transparently filtering, naming and labeling
//resources that are managed by the CLI.
type Visor struct {
	prefix string
	ns     string
	api    kubernetes.Interface
	logs   Logger
}

//NewVisor will setup a Kubernetes visor
func NewVisor(ns, prefix string, api kubernetes.Interface, logs Logger) *Visor {
	return &Visor{prefix, ns, api, logs}
}

//CreateResource will use the kube RESTClient to create a resource while using the context, adding the
//Nerd prefix and handling errors specific to our domain.
func (k *Visor) CreateResource(ctx context.Context, t KubeResourceType, v KubeManagedNames, name string) (err error) {
	vv, ok := v.(runtime.Object)
	if !ok {
		return errors.Errorf("provided value was not castable to runtime.Object")
	}

	genfix := "x-"
	var c rest.Interface
	switch t {
	case KubeResourceTypeJobs:
		c = k.api.BatchV1().RESTClient()
		genfix = "j-"

	default:
		return errors.Errorf("unknown Kubernetes resource type provided: '%s'", t)
	}

	if name != "" {
		v.SetName(k.prefix + name)
	} else {
		v.SetGenerateName(k.prefix + genfix)
	}

	labels := v.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	labels["nerd-app"] = "cli"
	v.SetLabels(labels)

	k.logs.Debugf("creating %s '%s' in namespace '%s' and labels '%v': %s", t, v.GetName(), k.ns, labels, ctx)
	err = c.Post().
		Namespace(k.ns).
		Resource(string(t)).
		Body(vv).
		Context(ctx).
		Do().
		Into(vv)

	if err != nil {
		return k.tagError(err)
	}

	v.SetName(strings.TrimPrefix(v.GetName(), k.prefix)) //normalize back to unprefixed resource name
	return nil
}

//ListResources will use the RESTClient to list resources while using the context and transparently
//filter resources managed by the CLI
func (k *Visor) ListResources(ctx context.Context, t KubeResourceType, v KubeListTranformer) (err error) {
	vv, ok := v.(runtime.Object)
	if !ok {
		return errors.Errorf("provided value was not castable to runtime.Object")
	}

	var c rest.Interface
	switch t {
	case KubeResourceTypeJobs:
		c = k.api.BatchV1().RESTClient()
	default:
		return errors.Errorf("unknown Kubernetes resource type provided: '%s'", t)
	}

	err = c.Get().
		Namespace(k.ns).
		VersionedParams(&metav1.ListOptions{LabelSelector: "nerd-app=cli"}, scheme.ParameterCodec).
		Resource(string(t)).
		Context(ctx).
		Do().
		Into(vv)

	if err != nil {
		return k.tagError(err)
	}

	//transform each managed item to return unprefixed
	v.Transform(func(in KubeManagedNames) KubeManagedNames {
		in.SetName(strings.TrimPrefix(in.GetName(), k.prefix))
		return in
	})

	return nil
}

func (k *Visor) tagError(err error) error {
	if uerr, ok := err.(*url.Error); ok && uerr.Err == context.DeadlineExceeded {
		return errDeadline{uerr}
	}

	if serr, ok := err.(*kuberr.StatusError); ok {
		if kuberr.IsAlreadyExists(serr) {
			return errAlreadyExists{err}
		}

		if kuberr.IsNotFound(serr) {
			details := serr.ErrStatus.Details
			if details.Kind == "namespaces" {
				return errNamespaceNotExists{err}
			}
		}

		if kuberr.IsInvalid(serr) {
			details := serr.ErrStatus.Details
			for _, cause := range details.Causes {
				if cause.Field == "metadata.name" {
					return errInvalidName{err}
				}
			}
		}
	}

	return errKubernetes{err} //generic kubernetes error
}
