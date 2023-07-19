package node

import (
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/component-helpers/apimachinery/lease"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
	"time"
)

const (
	LeaseDurationSeconds = 40
	LeaseNameSpace       = "kube-node-lease"
)

// StartLeaseController 启动lease租约控制器
func StartLeaseController(client kubernetes.Interface, nodeName string) {
	myclock := clock.RealClock{}

	renewInterval := time.Duration(LeaseDurationSeconds * 0.25)
	heartbeatFailure := func() {
		// 续租失败的清理工作
	}
	klog.Infoln("starting lease controller")

	ctl := lease.NewController(myclock, client, nodeName, LeaseDurationSeconds,
		heartbeatFailure, renewInterval, LeaseNameSpace, SetNodeOwnerFunc(client, nodeName))
	ctl.Run(wait.NeverStop)
}
