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

	if cs.State.Waiting != nil {
		return fmt.Sprintf("%s\n   └─ ⚠️  状态: Waiting | 原因: %s | 信息: %s",
			prefix, cs.State.Waiting.Reason, cs.State.Waiting.Message)
	}

	if cs.State.Terminated != nil {
		return fmt.Sprintf("%s\n   └─ 🛑 状态: Terminated | 原因: %s | 退出码: %d | 信息: %s",
			prefix, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode, cs.State.Terminated.Message)
	}

	// Running
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
