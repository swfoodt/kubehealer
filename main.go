package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// 1. 获取 kubeconfig 路径
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		// 假设 kubeconfig 文件在 ~/.kube/config
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		log.Fatal("无法找到 Home 目录，请手动设置 kubeconfig 路径")
	}

	// 2. 使用 kubeconfig 构建配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("无法加载 kubeconfig: %v", err)
	}

	// 3. 创建 Kubernetes 客户端集（Clientset）
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("无法创建 Clientset: %v", err)
	}

	// 4. 调用 API：列出 default 命名空间下的所有 Pod
	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("列出 Pod 失败: %v", err)
	}

	fmt.Println("--- 🚀 成功连接集群，列出 default 命名空间下的 Pod ---")
	if len(pods.Items) == 0 {
		fmt.Println("default 命名空间当前没有 Pod。")
	} else {
		fmt.Println("--- 🚀 Pod 状态详情 ---")
		for _, pod := range pods.Items {
			// 1. 获取 Pod 状态
			status := string(pod.Status.Phase) // Status.Phase: Running, Pending, Failed, etc.

			// 2. 获取节点名
			nodeName := pod.Spec.NodeName // 调度到的节点名称

			// 3. 获取重启次数 (只取第一个容器)
			var restartCount int32 = 0
			if len(pod.Status.ContainerStatuses) > 0 {
				restartCount = pod.Status.ContainerStatuses[0].RestartCount
			}

			fmt.Printf("Pod: %s, Status: %s, Node: %s, Restarts: %d\n",
				pod.Name, status, nodeName, restartCount)
		}
	}
}
