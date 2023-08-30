package lib

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/pkg/errors"
	certificatesv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	apiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/certificate/csr"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"log"
	"mykubelet/pkg/common"
	"os"
	"sigs.k8s.io/yaml"
	"time"
)

const (
	BootstrapPrivateKey = "./.kube/kubelet.key"
	BootstrapPem        = "./.kube/kubelet.pem"
	PrivateKeyName      = "kubelet.key"
	PemName             = "kubelet.pem"
)

// CreateCsr 创建csr资源
func CreateCsr(client *kubernetes.Clientset, nodeName string) (*certificatesv1.CertificateSigningRequest, error) {
	csrPem, err := GenCsrPem(nodeName)
	if err != nil {
		return nil, err
	}

	csrObj := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request: csrPem,
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageClientAuth,
				// 自动批复所需的许可密钥用途
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageKeyEncipherment,
			},
			ExpirationSeconds: pointer.Int32(int32(time.Second * 3600 / time.Second)),
			// SignerName:        certificatesv1.KubeAPIServerClientSignerName,	  // 手动批复
			SignerName: certificatesv1.KubeAPIServerClientKubeletSignerName, // 自动批复
		},
	}
	csrRet, err := client.CertificatesV1().CertificateSigningRequests().
		Create(context.Background(), csrObj, metav1.CreateOptions{})
	return csrRet, err
}

// GenCsrPem 生成csr证书请求文件
func GenCsrPem(nodeName string) ([]byte, error) {
	// 生成客户端私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// 保存私钥到文件
	if err = savePrivateKey(privateKey); err != nil {
		return nil, err
	}

	// 生成csr
	cr := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("system:node:%s", nodeName),
			Organization: []string{"system:nodes"},
		},
	}
	csrPem, err := cert.MakeCSRFromTemplate(privateKey, cr)
	if err != nil {
		return nil, err
	}

	return csrPem, nil
}

// 保存私钥到文件
func savePrivateKey(key *ecdsa.PrivateKey) error {
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	privatePem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: b,
		},
	)

	_ = os.Remove(BootstrapPrivateKey)
	err = os.WriteFile(BootstrapPrivateKey, privatePem, 0600)
	if err != nil {
		return err
	}

	return nil
}

// WaitForCsrApprove 等待csr批复，保存kubelet证书到文件
func WaitForCsrApprove(client *kubernetes.Clientset, csrObj *certificatesv1.CertificateSigningRequest) error {
	return waitForCsrApprove(client, csrObj)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3600)
	defer cancel()
	klog.Info("waiting for csr is approved...")

	csrData, err := csr.WaitForCertificate(ctx, client, csrObj.Name, csrObj.UID)
	if err != nil {
		klog.V(3).ErrorS(err, "approved timeout")
		return err
	}

	return os.WriteFile(BootstrapPem, csrData, 0600)
}

// 手动提取csr.WaitForCertificate()的核心代码，实现等待批准csr
// 核心方法：watchtools.UntilWithSync()
func waitForCsrApprove(client *kubernetes.Clientset, csrObj *certificatesv1.CertificateSigningRequest) error {
	var certData []byte
	lw := cache.NewListWatchFromClient(client.CertificatesV1().RESTClient(), "certificatesigningrequests", "", fields.Everything())

	_, err := watchtools.UntilWithSync(
		context.Background(),
		lw,
		&certificatesv1.CertificateSigningRequest{},
		nil,
		func(event watch.Event) (bool, error) {
			log.Println("进入了")
			if getCsr, ok := event.Object.(*certificatesv1.CertificateSigningRequest); ok {
				if getCsr.UID != csrObj.UID {
					return false, fmt.Errorf("csr %q changed UIDs", getCsr.Name)
				}
				approved := false
				for _, c := range getCsr.Status.Conditions {
					if c.Type == certificatesv1.CertificateDenied {
						return false, fmt.Errorf("certificate signing request is denied, reason: %v, message: %v", c.Reason, c.Message)
					}
					if c.Type == certificatesv1.CertificateFailed {
						return false, fmt.Errorf("certificate signing request failed, reason: %v, message: %v", c.Reason, c.Message)
					}
					if c.Type == certificatesv1.CertificateApproved {
						approved = true
					}
				}

				if approved {
					if len(getCsr.Status.Certificate) > 0 {
						log.Println("批准了")
						certData = getCsr.Status.Certificate
						return true, nil
					}
					klog.V(2).Infof("certificate signing request %s is approved, waiting to be issued", getCsr.Name)
				}
			}

			return false, nil
		},
	)
	if err != nil {
		klog.Fatalln(err)
	}
	return os.WriteFile(BootstrapPem, certData, 0600)
}

// GenKubeconfig 生成kubeconfig
func GenKubeconfig(masterUrl string) error {
	// 构建Config对象
	cfg := apiv1.Config{}
	cfg.APIVersion = "v1"
	cfg.Kind = "Config"
	contextName := "default-context"
	clusterName := "default-cluster"
	authName := "default-auth"

	cfg.Clusters = []apiv1.NamedCluster{
		{
			Name: clusterName,
			Cluster: apiv1.Cluster{
				Server:               masterUrl,
				CertificateAuthority: common.CAName,
			},
		},
	}
	cfg.Contexts = []apiv1.NamedContext{
		{
			Name: contextName,
			Context: apiv1.Context{
				Cluster:  clusterName,
				AuthInfo: authName,
			},
		},
	}
	cfg.AuthInfos = []apiv1.NamedAuthInfo{
		{
			Name: authName,
			AuthInfo: apiv1.AuthInfo{
				ClientCertificate: PemName,
				ClientKey:         PrivateKeyName,
			},
		},
	}
	cfg.CurrentContext = contextName

	// 生成yaml
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	klog.Infoln("writing kubelet-config to ", common.KubeConfig)
	return os.WriteFile(common.KubeConfig, b, 0600)
}

// NeedRequestCsr 是否需要请求csr证书
func NeedRequestCsr() bool {
	if _, err := os.Stat(common.KubeConfig); errors.Is(err, os.ErrNotExist) {
		return true
	}

	return false
}
