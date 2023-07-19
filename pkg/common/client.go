package common

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"net/url"
)

const (
	CAFile     = "./.kube/ca.crt"
	CAName     = "ca.crt"
	KubeConfig = "./.kube/kubeconfig"
)

// NewForBootstrapToken 根据token创建低权限的client
func NewForBootstrapToken(token, masterUrl string) *kubernetes.Clientset {
	urlObj, err := url.Parse(masterUrl)
	if err != nil {
		klog.Fatalln(err)
	}

	restConfig := &rest.Config{
		BearerToken: token,
		Host:        urlObj.Host,
		APIPath:     urlObj.Path,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
			CAFile:   CAFile,
		},
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Fatalln(err)
	}

	klog.V(3).Info("create clientset by bootstrap token")
	return client
}

// NewForKubeletConfig 根据kubeconfig创建clientset
func NewForKubeletConfig() *kubernetes.Clientset {
	restConfig, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
	if err != nil {
		klog.Fatalln(err)
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Fatalln(err)
	}

	return client
}
