package k8s

import (
	"flag"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client 封装了 Kubernetes 客户端集
type Client struct {
	Clientset *kubernetes.Clientset
}

// NewClient 初始化并返回一个 K8s 客户端
func NewClient() (*Client, error) {
	var kubeconfig *string

	// 尝试寻找 kubeconfig 文件
	if home := homedir.HomeDir(); home != "" {
		path := filepath.Join(home, ".kube", "config")
		kubeconfig = flag.String("kubeconfig", path, "(optional) absolute path to the kubeconfig file")
	} else {
		path := ""
		kubeconfig = &path
	}

	// 注意：这里为了兼容 Cobra，我们暂时手动解析一下 flag，或者直接构建配置
	// 在实际 CLI 中，通常由 Cobra 处理 flag，这里我们简化处理，直接读取默认路径
	// 如果想要更严谨，可以使用 clientcmd.BuildConfigFromFlags("", *kubeconfig)
	// 但为了确保在这一步不出错，我们直接加载:

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Clientset: clientset,
	}, nil
}
