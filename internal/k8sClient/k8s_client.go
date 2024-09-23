package k8sClient

import (
	"errors"
	"fmt"

	localConfig "github.com/stein-solutions/aks-spot-instance-tolerator/internal/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClientInterface interface {
	Clientset() kubernetes.Interface
}

type K8sClient struct {
	_clientset kubernetes.Interface
}

type ConfigProvider interface {
	BuildConfigFromFlags(string, string) (*rest.Config, error)
}

type DefaultConfigProvider struct {
}

func (c *DefaultConfigProvider) BuildConfigFromFlags(masterurl, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags(masterurl, kubeconfigPath)
}

func (k *K8sClient) Clientset() kubernetes.Interface {
	return k._clientset
}

func NewK8sClientDefault() K8sClientInterface {
	configProviderImpl := &DefaultConfigProvider{}
	return NewK8sClient(configProviderImpl)
}

func NewK8sClient(provider ConfigProvider) K8sClientInterface {
	config, err := rest.InClusterConfig()
	if err != nil && !errors.Is(err, rest.ErrNotInCluster) {
		return nil
	}
	if errors.Is(err, rest.ErrNotInCluster) {
		fmt.Println("Not running in cluster")
		//not running in cluster
		env := localConfig.NewConfig()
		fmt.Println("Trying with local kubeconfig: ", env.KubeConfig)
		config, err = provider.BuildConfigFromFlags("", env.KubeConfig)
		if err != nil {
			fmt.Println("Error building kubeconfig: ", err)
			return nil
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil
	}

	k8sClient := &K8sClient{
		_clientset: clientset,
	}

	return k8sClient
}
