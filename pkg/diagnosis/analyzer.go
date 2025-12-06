package diagnosis

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Analyzer struct {
	client *kubernetes.Clientset
}

func NewAnalyzer(client *kubernetes.Clientset) *Analyzer {
	return &Analyzer{
		client: client,
	}
}

// AnalyzePod 编排诊断流程
func (a *Analyzer) AnalyzePod(pod *corev1.Pod) {
	// 1. 获取并打印基础信息
	info := a.GetPodBasicInfo(pod)
	fmt.Println(info)

	// 2. 获取并打印容器状态
	fmt.Println("   --- 容器详情 ---")
	for _, cs := range pod.Status.ContainerStatuses {
		statusMsg := a.GetContainerStatus(cs)
		fmt.Println(statusMsg)
	}
}

// GetPodBasicInfo 提取基础信息字符串
func (a *Analyzer) GetPodBasicInfo(pod *corev1.Pod) string {
	return fmt.Sprintf("📦 Pod: %s | 命名空间: %s | 节点: %s\n   状态: %s | 重启总数: %d",
		pod.Name, pod.Namespace, pod.Spec.NodeName,
		pod.Status.Phase, sumRestarts(pod))
}

// GetContainerStatus 解析单个容器状态
func (a *Analyzer) GetContainerStatus(cs corev1.ContainerStatus) string {
	prefix := fmt.Sprintf("   ├─ 容器: %s", cs.Name)

	// Waiting 状态处理
	if cs.State.Waiting != nil {
		reason := cs.State.Waiting.Reason
		msg := cs.State.Waiting.Message

		// 镜像拉取失败的专门诊断
		if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
			return fmt.Sprintf("%s\n   └─ 🚫 镜像拉取失败: 无法获取镜像 '%s'\n      可能原因: 镜像名拼写错误 / 镜像不存在 / 私有仓库缺少 ImagePullSecrets\n      原始报错: %s",
				prefix, cs.Image, msg)
		}

		output := fmt.Sprintf("%s\n   └─ ⚠️  状态: Waiting | 原因: %s | 信息: %s",
			prefix, reason, msg)

		// 查看上次退出原因
		if cs.LastTerminationState.Terminated != nil {
			lastState := cs.LastTerminationState.Terminated
			exitInfo := explainExitCode(lastState.ExitCode)
			output += fmt.Sprintf("\n      👀 上次退出: %s | 退出码: %s",
				lastState.Reason, exitInfo)
		}

		return output
	}
	// Terminated 状态处理
	if cs.State.Terminated != nil {
		// 使用 explainExitCode 翻译退出码
		exitInfo := explainExitCode(cs.State.Terminated.ExitCode)

		return fmt.Sprintf("%s\n   └─ 🛑 状态: Terminated | 原因: %s | 退出码: %s | 信息: %s",
			prefix, cs.State.Terminated.Reason, exitInfo, cs.State.Terminated.Message)
	}

	// Running 状态处理
	status := fmt.Sprintf("%s\n   └─ ✅ 状态: Running", prefix)
	if cs.RestartCount > 0 {
		status += fmt.Sprintf(" (但已重启 %d 次)", cs.RestartCount)
	}
	return status
}

func sumRestarts(pod *corev1.Pod) int32 {
	var count int32
	for _, cs := range pod.Status.ContainerStatuses {
		count += cs.RestartCount
	}
	return count
}

// GetPodEvents 获取并打印 Pod 的相关事件
func (a *Analyzer) GetPodEvents(pod *corev1.Pod) {
	fmt.Println("   --- 📋 最近事件 (Events) ---")

	// 使用 FieldSelector 过滤出涉及该 Pod 的事件
	// involvedObject.uid = Pod UID (更精确，防止同名冲突)
	selector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.uid=%s",
		pod.Name, pod.Namespace, pod.UID)

	events, err := a.client.CoreV1().Events(pod.Namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: selector,
	})

	if err != nil {
		fmt.Printf("   ❌ 获取事件失败: %v\n", err)
		return
	}

	if len(events.Items) == 0 {
		fmt.Println("   (无事件记录)")
		return
	}

	// 按时间排序 (LastTimestamp)
	sort.Slice(events.Items, func(i, j int) bool {
		return events.Items[i].LastTimestamp.Time.Before(events.Items[j].LastTimestamp.Time)
	})

	// 打印最近的 5 条
	start := 0
	if len(events.Items) > 5 {
		start = len(events.Items) - 5
	}

	for i := start; i < len(events.Items); i++ {
		e := events.Items[i]
		age := translateTimestamp(e.LastTimestamp.Time)

		icon := "🔹"
		if e.Type == "Warning" {
			icon = "🔸"
		}

		fmt.Printf("   %s [%s] %s: %s\n", icon, age, e.Reason, e.Message)
	}
}

// translateTimestamp 计算时间差
func translateTimestamp(t time.Time) string {
	if t.IsZero() {
		return "未知"
	}
	duration := time.Since(t)
	if duration.Seconds() < 60 {
		return fmt.Sprintf("%.0f秒前", duration.Seconds())
	}
	if duration.Minutes() < 60 {
		return fmt.Sprintf("%.0f分钟前", duration.Minutes())
	}
	return fmt.Sprintf("%.0f小时前", duration.Hours())
}

// 常见退出码映射表
var exitCodeMap = map[int32]string{
	0:   "Completed (正常退出)",
	1:   "General Error (应用内部错误)",
	2:   "Misuse of Shell Builtins (Shell内建命令误用)",
	126: "Invoked Command Cannot Execute (命令不可执行)",
	127: "Command Not Found (命令未找到)",
	128: "Invalid Exit Argument (无效的退出参数)",
	130: "Script Terminated by Control-C (被Ctrl+C终止)",
	137: "SIGKILL (强制终止/OOMKilled - 内存溢出)",
	143: "SIGTERM (优雅终止)",
}

// explainExitCode 将数字退出码转换为人类可读的字符串
func explainExitCode(code int32) string {
	if msg, ok := exitCodeMap[code]; ok {
		return fmt.Sprintf("%d (%s)", code, msg)
	}

	// 处理 128+n 的信号退出情况
	if code > 128 {
		return fmt.Sprintf("%d (Signal %d)", code, code-128)
	}

	return fmt.Sprintf("%d (未知错误码)", code)
}
