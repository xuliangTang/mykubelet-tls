package bootstrap

import (
	"k8s.io/klog/v2"
	"mykubelet/pkg/bootstrap/lib"
	"mykubelet/pkg/common"
)

// BootStrap 启动引导
// 加入节点前，使用TLS Bootstrap的自动化机制，请求apiServer时自动签发证书
// 1. kubelet先使用一个预先商定好的低权限token连接到kube-apiserver
// 2. 向kube-apiserver申请证书，然后kube-controller-manager给kubelet动态签署证书（包括手动批准CSR）
// 3. 后续kubelet都将通过动态签署的证书与kube-apiserver通信
func BootStrap(nodeName, masterUrl string) {
	// 判断是否已经存在kubeconfig
	if !lib.NeedRequestCsr() {
		klog.Infoln("kubelet.config already exists. skip csr-boot")
		return
	}

	// token是提交yaml由apiserver生成的， 不应该在kubelet客户端生成token
	//homeClient := common.NewForHomeConfig()
	//token, err := lib.GenToken(homeClient)
	//if err != nil {
	//	klog.Fatalln(err)
	//}

	token := ""

	// 根据token创建低权限的client
	klog.Infoln("begin bootstrap")
	bootClient := common.NewForBootstrapToken(token, masterUrl)

	// 授权kubelet创建csr（证书签名请求）
	csrObj, err := lib.CreateCsr(bootClient, nodeName)
	if err != nil {
		klog.Fatalln(err)
	}

	// 等待批复
	if err = lib.WaitForCsrApprove(bootClient, csrObj); err != nil {
		klog.Fatalln(err)
	}
	klog.Infoln("kubelet pem-files have been saved in .kube")

	// 获取证书生成kubeconfig
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
