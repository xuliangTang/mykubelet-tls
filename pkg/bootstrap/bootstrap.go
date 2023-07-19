package bootstrap

import (
	"k8s.io/klog/v2"
	"mykubelet/pkg/bootstrap/lib"
	"mykubelet/pkg/common"
)

func BootStrap(token, nodeName, masterUrl string) {
	// 判断是否已经存在kubeconfig
	if !lib.NeedRequestCsr() {
		klog.Infoln("kubelet.config already exists. skip csr-boot")
		return
	}

	klog.Infoln("begin bootstrap")
	bootClient := common.NewForBootstrapToken(token, masterUrl)
	csrObj, err := lib.CreateCsr(bootClient, nodeName)
	if err != nil {
		klog.Fatalln(err)
	}

	// 等待批复
	if err = lib.WaitForCsrApprove(bootClient, csrObj); err != nil {
		klog.Fatalln(err)
	}
	klog.Infoln("kubelet pem-files have been saved in .kube")

	// 生成kubeconfig
	if err = lib.GenKubeconfig(masterUrl); err != nil {
		klog.Fatalln(err)
	}

	client := common.NewForKubeletConfig()
	info, err := client.ServerVersion()
	if err != nil {
		klog.Fatalln(err)
	}
	klog.Infoln(info.String())
}
