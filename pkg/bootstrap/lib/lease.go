package lib

import (
	"context"
	"encoding/json"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"log"
	"time"
)

const (
	leaseNS   = "kube-node-lease"
	leaseName = "mylain"
)

var lease *coordinationv1.Lease

// Renew lease续期
func Renew(client *kubernetes.Clientset) {
	// 获取节点的lease对象
	getLease, err := client.CoordinationV1().Leases(leaseNS).
		Get(context.Background(), leaseName, metav1.GetOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	lease = getLease

	// 模拟kubelet心跳传递给apiServer，每隔 NodeLeaseDurationSeconds*0.25s 续租一次（40*0.25=10s）
	// controller-manager会监控节点状态，检查租约是否过期，过期会更新node状态为unknown
	leaseDuration := time.Duration(40) * time.Second
	renewInterval := time.Duration(float64(leaseDuration) * 0.25)

	go func() {
		for {
			renewLease(client)
			time.Sleep(renewInterval)
		}
	}()
}

// lease续租 更新spec.renewTime
func renewLease(client *kubernetes.Clientset) {
	now := metav1.NewMicroTime(time.Now())
	lease.Spec.RenewTime = &now
	newLease, err := client.CoordinationV1().Leases(leaseNS).Update(context.Background(), lease, metav1.UpdateOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	lease = newLease
}

type Value struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}
type Cond struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value *Value `json:"value"`
}

// SetNodeReady 更新节点的ready为true
func SetNodeReady(client *kubernetes.Clientset) {
	payload := []Cond{
		{
			Op:   "replace",
			Path: "/status/conditions/3",
			Value: &Value{
				Type:   "Ready",
				Status: "True",
			},
		},
	}
	payloadJson, _ := json.Marshal(payload)

	_, err := client.CoreV1().Nodes().Patch(context.Background(), leaseName, types.JSONPatchType,
		payloadJson, metav1.PatchOptions{}, "status")
	if err != nil {
		log.Fatalln(err)
	}
}
