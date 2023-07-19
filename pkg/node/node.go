package node

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"runtime"
)

// RegisterNode 注册节点
func RegisterNode(client *kubernetes.Clientset, nodeName string) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				corev1.LabelHostname:   nodeName,
				corev1.LabelOSStable:   runtime.GOOS,
				corev1.LabelArchStable: runtime.GOARCH,
			},
		},
		Spec: corev1.NodeSpec{},
	}
	_, err := client.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	if err != nil {
		klog.Fatalln(err)
	}
}
