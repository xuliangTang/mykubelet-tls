package main

import (
	"flag"
	"k8s.io/klog/v2"
	"mykubelet/pkg/bootstrap"
	"mykubelet/pkg/common"
	"mykubelet/pkg/node"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// bootstrap认证生成kubelet config
	masterUrl := "https://110.41.142.160:6443"
	nodeName := "mykubelet"
	token := "tttghq.s5uhy3h2gz0cskxg"
	bootstrap.BootStrap(token, nodeName, masterUrl)

	// 注册节点
	client := common.NewForKubeletConfig()
	node.RegisterNode(client, nodeName)

	// 启动租约控制器
	node.StartLeaseController(client, nodeName)
}
