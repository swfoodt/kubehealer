package main

import (
	"fmt"
	"os"

	"context"

	"github.com/swfoodt/kubehealer/pkg/diagnosis"
	"github.com/swfoodt/kubehealer/pkg/k8s"
	"github.com/swfoodt/kubehealer/pkg/report"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 定义变量存储输出格式
var outputFormat string

// diagnoseCmd 代表 diagnose 命令
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose [pod-name]",
	Short: "诊断指定的 Pod",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		podName := args[0]

		// 只有在默认模式下才打印这行，否则会污染 Markdown 输出
		if outputFormat == "" || outputFormat == "table" {
			fmt.Printf("🔍 正在诊断 Pod: %s ...\n\n", podName)
		}

		// 初始化客户端
		client, err := k8s.NewClient()
		if err != nil {
			fmt.Printf("❌ 错误: 无法连接集群 - %v\n", err)
			os.Exit(1)
		}

		// 获取 Pod
		pod, err := client.Clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("❌ 错误: 无法找到 Pod %s - %v\n", podName, err)
			os.Exit(1)
		}

		// 调用分析器
		analyzer := diagnosis.NewAnalyzer(client.Clientset)
		result := analyzer.AnalyzePod(pod)

		// 3. 根据参数选择输出
		switch outputFormat {
		case "md", "markdown":
			md := report.GenerateMarkdown(result)
			fmt.Println(md)
		case "json":
			// Day 18 会做这个，先占位
			fmt.Println("JSON output coming soon in Day 18")
		default:
			// 默认打印表格
			report.PrintTable(result)
		}

		// 打印 PID 和程序退出标记
		fmt.Printf("\n🏁 [PID: %d] 诊断结束，程序即将退出。\n", os.Getpid())

		// 强制写入新行，清除终端残留输入/输出
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)

	// 2. 绑定参数 --output 或 -o
	diagnoseCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "输出格式 (table, md, json)")
}
