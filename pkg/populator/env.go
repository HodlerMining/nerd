package populator

import (
	"os"
	"sync/atomic"

	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd/api"
)

type EnvPopulator struct {
	// kubeConfigFile is the path where the kube config is stored
	// Only access this with atomic ops
	kubeConfigFile atomic.Value
}

func newEnv(kubeConfigFile string) *EnvPopulator {
	e := &EnvPopulator{}
	e.kubeConfigFile.Store(kubeConfigFile)
	return e
}

func (e *EnvPopulator) GetKubeConfigFile() string {
	return e.kubeConfigFile.Load().(string)
}

// PopulateKubeConfig populates the kube config file with the info found in the environment.
func (e *EnvPopulator) PopulateKubeConfig(project string) error {
	cluster := api.NewCluster()
	cluster.Server = os.Getenv("KUBE_CLUSTER_ADDR")

	// user
	user := api.NewAuthInfo()
	user.Token = os.Getenv("KUBE_TOKEN")

	// context
	context := api.NewContext()
	context.Cluster = project
	context.AuthInfo = project
	context.Namespace = os.Getenv("KUBE_NAMESPACE")

	// read existing config or create new if does not exist
	kubecfg, err := ReadConfigOrNew(e.GetKubeConfigFile())
	if err != nil {
		return err
	}
	kubecfg.Clusters[project] = cluster
	kubecfg.AuthInfos[project] = user
	kubecfg.CurrentContext = project
	kubecfg.Contexts[project] = context

	// write back to disk
	if err := WriteConfig(kubecfg, e.GetKubeConfigFile()); err != nil {
		return errors.Wrap(err, "writing kubeconfig")
	}

	return nil
}
