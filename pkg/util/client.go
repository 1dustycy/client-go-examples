package util

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientInterface .
type ClientInterface interface {
	kubernetes.Interface
	dynamic.Interface
}

// NewClientSet .
func NewClientSet(contextName string) (cli ClientInterface, err error) {
	conf, err := newConfig(contextName)
	if err != nil {
		return
	}

	var set clientSet

	if set.Clientset, err = kubernetes.NewForConfig(conf); err != nil {
		return
	}
	if set.dynamicClient, err = dynamic.NewForConfig(conf); err != nil {
		return
	}

	return set, err
}

func newConfig(contextName string) (conf *rest.Config, err error) {
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.
		NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).
		ClientConfig()
}

type clientSet struct {
	*kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// Resource .
func (c clientSet) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return c.dynamicClient.Resource(resource)
}
