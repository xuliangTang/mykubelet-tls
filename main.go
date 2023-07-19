package main

import (
	"flag"
	"k8s.io/klog/v2"
	"mykubelet/pkg/bootstrap"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	masterUrl := "https://110.41.142.160:6443"
	token := "tttghq.s5uhy3h2gz0cskxg"
	bootstrap.BootStrap(token, "mytest", masterUrl)
}
