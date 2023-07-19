package node

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"runtime"
)

// 设置节点status信息
func setNodeStatus(node *corev1.Node) {
	node.Status.NodeInfo = nodeInfo()
	node.Status.DaemonEndpoints = nodeDaemonEndpoints(10250)
	node.Status.Addresses = nodeAddresses()
	node.Status.Conditions = nodeConditions()
	node.Status.Capacity = nodeCapacity()
}

// 节点信息
func nodeInfo() corev1.NodeSystemInfo {
	return corev1.NodeSystemInfo{
		KubeletVersion: "v1.22.99",
	}
}

// 节点端口
func nodeDaemonEndpoints(port int32) corev1.NodeDaemonEndpoints {
	return corev1.NodeDaemonEndpoints{
		KubeletEndpoint: corev1.DaemonEndpoint{
			Port: port,
		},
	}
}

// 节点的内部IP
func nodeAddresses() []corev1.NodeAddress {
	return []corev1.NodeAddress{
		{
			Type:    "InternalIP",
			Address: "121.231.134.231",
		},
	}
}

// 节点状态集合
func nodeConditions() []corev1.NodeCondition {
	return []corev1.NodeCondition{
		{
			Type:               "Ready",
			Status:             corev1.ConditionTrue,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletReady",
			Message:            "kubelet is ready.",
		},
		{
			Type:               "OutOfDisk",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientDisk",
			Message:            "kubelet has sufficient disk space available",
		},
		{
			Type:               "MemoryPressure",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientMemory",
			Message:            "kubelet has sufficient memory available",
		},
		{
			Type:               "DiskPressure",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasNoDiskPressure",
			Message:            "kubelet has no disk pressure",
		},
		{
			Type:               "NetworkUnavailable",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "RouteCreated",
			Message:            "RouteController created a route",
		},
	}
}

// 节点资源信息 CPU和内存
func nodeCapacity() corev1.ResourceList {
	var cpuQ resource.Quantity
	cpuQ.Set(int64(runtime.NumCPU()))

	var memQ resource.Quantity
	memQ.Set(int64(1024 * 1024 * 1024 * 32)) // 好比32G内存 假的
	memQ.Format = resource.BinarySI
	return corev1.ResourceList{
		"cpu":    cpuQ,
		"memory": memQ,
		"pods":   resource.MustParse("200"), // 最多创建多少个pod
	}
}
