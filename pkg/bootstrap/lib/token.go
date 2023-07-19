package lib

import (
	"context"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	"strings"
	"time"
)

const (
	nodeBootstrapTokenAuthGroup = "system:bootstrappers:kubeadm:default-node-token"
)

// GenToken 生成token
func GenToken(client *kubernetes.Clientset) (*BootstrapToken, error) {
	// 构建token对象
	bootstrapToken, err := newBootstrapTokenWithOptions()
	if err != nil {
		return nil, err
	}

	// 转为map
	bootstrapTokenMap := make(map[string][]byte)
	if err = mapstructure.WeakDecode(bootstrapToken, &bootstrapTokenMap); err != nil {
		return nil, err
	}

	// 构建secret
	secretObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapTokenSecretName(string(bootstrapToken.TokenId)),
			Namespace: metav1.NamespaceSystem,
		},
		Type: corev1.SecretTypeBootstrapToken,
		Data: bootstrapTokenMap,
	}

	// 创建
	_, err = client.CoreV1().Secrets(metav1.NamespaceSystem).
		Create(context.Background(), secretObj, metav1.CreateOptions{})
	return bootstrapToken, err
}

// returns the expected name for the Secret storing the
// Bootstrap Token in the Kubernetes API.
func bootstrapTokenSecretName(tokenID string) string {
	return fmt.Sprintf("%s%s", api.BootstrapTokenSecretPrefix, tokenID)
}

type BootstrapToken struct {
	TokenId                      []byte `mapstructure:"token-id"`
	TokenSecret                  []byte `mapstructure:"token-secret"`
	Description                  []byte `mapstructure:"description,omitempty"`
	Expires                      []byte `mapstructure:"expires"`
	UsageBootstrapAuthentication []byte `mapstructure:"usage-bootstrap-authentication"`
	UsageBootstrapSigning        []byte `mapstructure:"usage-bootstrap-signing"`
	AuthExtraGroups              []byte `mapstructure:"auth-extra-groups"`
}

func (this *BootstrapToken) GetToken() string {
	return fmt.Sprintf("%s.%s", this.TokenId, this.TokenSecret)
}

// 构建一个带默认参数的token
func newBootstrapTokenWithOptions() (*BootstrapToken, error) {
	// 使用内置包生成一个token
	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return nil, err
	}
	token := strings.Split(tokenStr, ".")
	if len(token) != 2 {
		return nil, errors.New("generate token error")
	}
	tokenId, tokenSecret := token[0], token[1]
	expire := time.Now().Add(time.Hour * 24)

	return &BootstrapToken{
		TokenId:                      []byte(tokenId),
		TokenSecret:                  []byte(tokenSecret),
		Expires:                      []byte(expire.UTC().Format(time.RFC3339)),
		UsageBootstrapAuthentication: []byte("true"),
		UsageBootstrapSigning:        []byte("true"),
		AuthExtraGroups:              []byte(nodeBootstrapTokenAuthGroup),
	}, nil
}
