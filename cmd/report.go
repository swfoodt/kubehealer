package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// reportCmd 代表 report 命令
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "管理历史诊断报告",
	Long:  `查看、打开或管理生成的 HTML 诊断报告。`,
}

// reportListCmd 代表 report list 子命令
var reportListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有历史报告",
	Run: func(cmd *cobra.Command, args []string) {
		reportDir := "reports"

		// 读取目录
		files, err := os.ReadDir(reportDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("📭 暂无历史报告 (reports 目录不存在)")
				return
			}
			fmt.Printf("❌ 读取目录失败: %v\n", err)
			return
		}

		// 准备表格数据
		var data [][]string
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".html") {
				continue
			}

			info, _ := file.Info()
			size := fmt.Sprintf("%.1f KB", float64(info.Size())/1024)
			modTime := info.ModTime().Format("2006-01-02 15:04:05")

			// 从文件名解析 Pod 名称
			// 格式: podname_report_timestamp.html
			name := file.Name()
			podName := "Unknown"
			if parts := strings.Split(name, "_report_"); len(parts) > 0 {
				podName = parts[0]
			}

			data = append(data, []string{modTime, podName, size, name})
		}

		if len(data) == 0 {
			fmt.Println("📭 暂无历史报告")
			return
		}

		// 渲染表格
		fmt.Printf("📂 历史报告列表 (%s):\n", reportDir)
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"生成时间", "Pod 名称", "大小", "文件名"})
		table.SetBorder(false)
		table.AppendBulk(data)
		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.AddCommand(reportListCmd)
}
