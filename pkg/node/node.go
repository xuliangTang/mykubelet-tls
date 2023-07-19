package node

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"runtime"
)

// RegisterNode 注册节点
func RegisterNode(client *kubernetes.Clientset, nodeName string) {
	nodeObj := &corev1.Node{
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

	getNode, err := client.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	// 创建节点
	if err != nil || getNode == nil {
		createdNode, err := client.CoreV1().Nodes().Create(context.Background(), nodeObj, metav1.CreateOptions{})
		if err != nil {
			klog.Fatalln(err)
		}
		klog.Infof("create node %s success", nodeName)
		getNode = createdNode
	}

	// 设置节点信息
	newNode := getNode.DeepCopy()
	setNodeStatus(newNode)

	// 获取策略性patch内容
	patchBytes, err := preparePatchBytesforNodeStatus(types.NodeName(nodeName), getNode, newNode)
	if err != nil {
		klog.Fatalln(err)
	}

	// patch更新
	_, err = client.CoreV1().Nodes().Patch(context.Background(), nodeName, types.StrategicMergePatchType,
		patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		klog.Fatalln(err)
	}
	klog.Infoln("node status update success")
}
