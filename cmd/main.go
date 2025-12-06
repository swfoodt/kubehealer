package main

import (
	"context"
	"fmt"
	"os"

	"github.com/swfoodt/kubehealer/pkg/k8s" // 确保这里是您的 module 名，即 go.mod 中的 module name

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// 定义根命令
	var rootCmd = &cobra.Command{
		Use:   "kubehealer",
		Short: "KubeHealer: K8s 容器层诊断工具",
		Long:  `一个基于 Go + client-go 的 Kubernetes 诊断工具，用于快速定位 Pod 异常。`,
	}

	// 定义 diagnose 子命令
	var diagnoseCmd = &cobra.Command{
		Use:   "diagnose [pod-name]",
		Short: "诊断指定的 Pod",
		Args:  cobra.ExactArgs(1), // 强制要求 1 个参数
		Run: func(cmd *cobra.Command, args []string) {
			podName := args[0]
			fmt.Printf("正在诊断 Pod: %s ...\n", podName)

			// 1. 初始化客户端
			client, err := k8s.NewClient()
			if err != nil {
				fmt.Printf("错误: 无法连接集群 - %v\n", err)
				os.Exit(1)
			}

			// 2. 获取 Pod 信息 (Day 2/3 的逻辑复现)
			pod, err := client.Clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("错误: 无法找到 Pod %s - %v\n", podName, err)
				os.Exit(1)
			}

			// 3. 简单输出结果 (验证连通性)
			fmt.Printf("✅ 找到 Pod: %s | 状态: %s | 节点: %s\n",
				pod.Name, pod.Status.Phase, pod.Spec.NodeName)
		},
	}

	// 将子命令加入根命令
	rootCmd.AddCommand(diagnoseCmd)

	// 执行
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
