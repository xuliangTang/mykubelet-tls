package node

import (
	"context"
	"fmt"
	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// 获取新旧节点的patch内容
// pkg/util/node/node.go   272行
func preparePatchBytesforNodeStatus(nodeName types.NodeName, oldNode *v1.Node, newNode *v1.Node) ([]byte, error) {
	oldData, err := json.Marshal(oldNode)
	if err != nil {
		return nil, fmt.Errorf("failed to Marshal oldData for node %q: %v", nodeName, err)
	}

	manuallyPatchAddresses := (len(oldNode.Status.Addresses) > 0) && !equality.Semantic.DeepEqual(oldNode.Status.Addresses, newNode.Status.Addresses)

	diffNode := newNode.DeepCopy()
	diffNode.Spec = oldNode.Spec
	if manuallyPatchAddresses {
		diffNode.Status.Addresses = oldNode.Status.Addresses
	}
	newData, err := json.Marshal(diffNode)
	if err != nil {
		return nil, fmt.Errorf("failed to Marshal newData for node %q: %v", nodeName, err)
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, v1.Node{})
	if err != nil {
		return nil, fmt.Errorf("failed to CreateTwoWayMergePatch for node %q: %v", nodeName, err)
	}
	if manuallyPatchAddresses {
		patchBytes, err = fixupPatchForNodeStatusAddresses(patchBytes, newNode.Status.Addresses)
		if err != nil {
			return nil, fmt.Errorf("failed to fix up NodeAddresses in patch for node %q: %v", nodeName, err)
		}
	}

	return patchBytes, nil
}

func fixupPatchForNodeStatusAddresses(patchBytes []byte, addresses []v1.NodeAddress) ([]byte, error) {
	var patchMap map[string]interface{}
	if err := json.Unmarshal(patchBytes, &patchMap); err != nil {
		return nil, err
	}

	addrBytes, err := json.Marshal(addresses)
	if err != nil {
		return nil, err
	}
	var addrArray []interface{}
	if err := json.Unmarshal(addrBytes, &addrArray); err != nil {
		return nil, err
	}
	addrArray = append(addrArray, map[string]interface{}{"$patch": "replace"})

	status := patchMap["status"]
	if status == nil {
		status = map[string]interface{}{}
		patchMap["status"] = status
	}
	statusMap, ok := status.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data in patch")
	}
	statusMap["addresses"] = addrArray

	return json.Marshal(patchMap)
}

// SetNodeOwnerFunc helps construct a newLeasePostProcessFunc which sets
// a node OwnerReference to the given lease object
func SetNodeOwnerFunc(c kubernetes.Interface, nodeName string) func(lease *coordinationv1.Lease) error {
	return func(lease *coordinationv1.Lease) error {
		// Setting owner reference needs node's UID. Note that it is different from
		// kubelet.nodeRef.UID. When lease is initially created, it is possible that
		// the connection between master and node is not ready yet. So try to set
		// owner reference every time when renewing the lease, until successful.
		if len(lease.OwnerReferences) == 0 {
			if node, err := c.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{}); err == nil {
				lease.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: v1.SchemeGroupVersion.WithKind("Node").Version,
						Kind:       v1.SchemeGroupVersion.WithKind("Node").Kind,
						Name:       nodeName,
						UID:        node.UID,
					},
				}
			} else {
				klog.ErrorS(err, "Failed to get node when trying to set owner ref to the node lease", "node", klog.KRef("", nodeName))
				return err
			}
		}
		return nil
	}
}
