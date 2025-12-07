package main

import (
	"fmt"
	"os"

	"context"

	"github.com/swfoodt/kubehealer/pkg/diagnosis"
	"github.com/swfoodt/kubehealer/pkg/k8s"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// diagnoseCmd 代表 diagnose 命令
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose [pod-name]",
	Short: "诊断指定的 Pod",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		podName := args[0]
		fmt.Printf("🔍 正在诊断 Pod: %s ...\n\n", podName)

		// 1. 初始化客户端
		client, err := k8s.NewClient()
		if err != nil {
			fmt.Printf("❌ 错误: 无法连接集群 - %v\n", err)
			os.Exit(1)
		}

		// 2. 获取 Pod
		pod, err := client.Clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("❌ 错误: 无法找到 Pod %s - %v\n", podName, err)
			os.Exit(1)
		}

		// 3. 调用分析器
		analyzer := diagnosis.NewAnalyzer(client.Clientset)
		result := analyzer.AnalyzePod(pod)

		// 4. 简单打印结果
		fmt.Printf("---------------------------------------------\n")
		fmt.Printf("Day 15 结构体验收: %s (Restarts: %d)\n", result.PodName, result.RestartCount)
		fmt.Printf("---------------------------------------------\n")

		for _, c := range result.Containers {
			fmt.Printf("容器: %s | 状态: %s\n", c.Name, c.State)
			if c.ResourceInfo != "" {
				fmt.Printf("  📊 资源: %s\n", c.ResourceInfo)
			}
			// 打印 Issue
			for _, issue := range c.Issues {
				fmt.Printf("  [%s] %s\n", issue.Type, issue.Title)
				if issue.Suggestion != "" {
					fmt.Printf("     💡 建议: %s\n", issue.Suggestion)
				}
			}
		}

		if len(result.Events) > 0 {
			fmt.Printf("\nEvents:\n")
			for _, e := range result.Events {
				fmt.Println("  " + e)
			}
		}

		// 打印 PID 和程序退出标记
		fmt.Printf("\n🏁 [PID: %d] 诊断结束，程序即将退出。\n", os.Getpid())

		// 强制写入新行，清除终端残留输入/输出
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
}
