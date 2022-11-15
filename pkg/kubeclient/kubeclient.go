package kubeclient

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func NewKubeClient() (*kubernetes.Clientset, dynamic.Interface, error) {
	// creates the in-cluster config
	var kubeconfig string
	config, err := rest.InClusterConfig()
	if err != nil {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
			kubeconfig = "/Users/zaller/Development/test-api-3/src/github.com/argoproj/metrics/kubeconfig"
		}
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset, dynamicClient, nil
}
