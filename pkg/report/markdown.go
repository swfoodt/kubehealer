package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/swfoodt/kubehealer/pkg/diagnosis"
)

// GenerateMarkdown 生成 Markdown 格式的诊断报告
func GenerateMarkdown(result diagnosis.DiagnosisResult) string {
	var sb strings.Builder

	// 标题与元数据
	sb.WriteString(fmt.Sprintf("# 🚑 KubeHealer 诊断报告: %s\n\n", result.PodName))
	sb.WriteString(fmt.Sprintf("> 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 基础信息表格
	sb.WriteString("## 1. 基础信息\n\n")
	sb.WriteString("| 指标 | 值 |\n")
	sb.WriteString("| :--- | :--- |\n")
	sb.WriteString(fmt.Sprintf("| **Pod 名称** | `%s` |\n", result.PodName))
	sb.WriteString(fmt.Sprintf("| **命名空间** | `%s` |\n", result.Namespace))
	sb.WriteString(fmt.Sprintf("| **所在节点** | `%s` |\n", result.NodeName))
	sb.WriteString(fmt.Sprintf("| **当前状态** | **%s** |\n", result.Phase))
	sb.WriteString(fmt.Sprintf("| **重启次数** | %d |\n\n", result.RestartCount))

	// 容器分析
	sb.WriteString("## 2. 容器深度分析\n\n")
	for _, c := range result.Containers {
		icon := "✅"
		if c.State != "Running" {
			icon = "⚠️"
		}
		// 如果有 Error 级别的 Issue
		for _, issue := range c.Issues {
			if issue.Type == "Error" {
				icon = "🛑"
				break
			}
		}

		sb.WriteString(fmt.Sprintf("### %s 容器: %s\n\n", icon, c.Name))
		sb.WriteString(fmt.Sprintf("- **状态**: %s\n", c.State))
		sb.WriteString(fmt.Sprintf("- **资源配置**: `%s`\n", strings.ReplaceAll(c.ResourceInfo, "\n", " ")))

		if c.Reason != "" {
			sb.WriteString(fmt.Sprintf("- **原因**: %s\n", c.Reason))
		}
		if c.Message != "" {
			sb.WriteString(fmt.Sprintf("- **详细信息**: %s\n", c.Message))
		}
		if c.ExitCode != 0 {
			sb.WriteString(fmt.Sprintf("- **退出码**: %d\n", c.ExitCode))
		}

		// 诊断建议区域
		if len(c.Issues) > 0 {
			sb.WriteString("\n**🔍 诊断发现:**\n\n")
			for _, issue := range c.Issues {
				prefix := "⚠️"
				if issue.Type == "Error" {
					prefix = "🛑"
				}
				sb.WriteString(fmt.Sprintf("> %s **%s**\n", prefix, issue.Title))
				if issue.RawError != "" {
					sb.WriteString(fmt.Sprintf("> *原始报错: %s*\n", issue.RawError))
				}
				if issue.Suggestion != "" {
					sb.WriteString(fmt.Sprintf("> **💡 修复建议**: %s\n", issue.Suggestion))
				}
				sb.WriteString(">\n") // 空行分隔
			}
		}
		sb.WriteString("\n---\n\n")
	}

	// 事件列表
	sb.WriteString("## 3. 最近事件 (Events)\n\n")
	if len(result.Events) == 0 {
		sb.WriteString("*暂无事件记录*\n")
	} else {
		for _, e := range result.Events {
			sb.WriteString(fmt.Sprintf("- %s\n", e))
		}
	}

	return sb.String()
}
