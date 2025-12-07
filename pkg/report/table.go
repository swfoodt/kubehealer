package report

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/swfoodt/kubehealer/pkg/diagnosis"
)

// PrintTable 将诊断结果渲染为终端表格
func PrintTable(result diagnosis.DiagnosisResult) {
	fmt.Println()
	printBasicInfo(result)
	fmt.Println()
	printContainerInfo(result)
	fmt.Println()
	printEvents(result)
	fmt.Println()
}

func printBasicInfo(result diagnosis.DiagnosisResult) {
	data := [][]string{
		{"Pod 名称", result.PodName},
		{"命名空间", result.Namespace},
		{"所在节点", result.NodeName},
		{"当前状态", result.Phase},
		{"重启总数", fmt.Sprintf("%d 次", result.RestartCount)},
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"基础信息", "值"})
	table.SetBorder(false)
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Normal},
	)
	table.AppendBulk(data)
	table.Render()
}

func printContainerInfo(result diagnosis.DiagnosisResult) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"容器", "状态", "资源配置", "诊断详情"})
	table.SetRowLine(true) // 显示行分割线

	for _, c := range result.Containers {
		// 构造诊断详情文本 (Reason + Message + Issues)
		var details []string

		// 1. 基础原因
		if c.Reason != "" {
			details = append(details, fmt.Sprintf("Reason: %s", c.Reason))
		}
		if c.Message != "" {
			// 如果 Message 太长，截断一下，或者换行显示
			details = append(details, fmt.Sprintf("Msg: %s", c.Message))
		}
		if c.ExitCode != 0 {
			details = append(details, fmt.Sprintf("ExitCode: %d", c.ExitCode))
		}

		// 2. 规则引擎发现的问题 (加粗/红色)
		for _, issue := range c.Issues {
			prefix := "⚠️"
			if issue.Type == "Error" {
				prefix = "🛑"
			}
			details = append(details, fmt.Sprintf("%s %s", prefix, issue.Title))
			if issue.Suggestion != "" {
				details = append(details, fmt.Sprintf("   💡 %s", issue.Suggestion))
			}
		}

		// 3. 资源信息简化显示
		resInfo := strings.ReplaceAll(c.ResourceInfo, " | ", "\n")

		table.Append([]string{
			c.Name,
			c.State,
			resInfo,
			strings.Join(details, "\n"),
		})
	}

	fmt.Println("📋 容器分析:")
	table.Render()
}

func printEvents(result diagnosis.DiagnosisResult) {
	if len(result.Events) == 0 {
		return
	}

	// 直接打印文本列表
	fmt.Println("🕒 最近事件 (Events):")
	for _, e := range result.Events {
		fmt.Println("  " + e)
	}
}
