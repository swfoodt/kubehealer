package main

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd 代表没有调用子命令时的基础命令
var rootCmd = &cobra.Command{
	Use:   "kubehealer",
	Short: "KubeHealer: K8s 容器层诊断工具",
	Long:  `一个基于 Go + client-go 的 Kubernetes 诊断工具，用于快速定位 Pod 异常。`,
	//在此处可以添加全局 flag，例如 --kubeconfig
}

// Execute 将所有子命令添加到根命令并设置标志
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
