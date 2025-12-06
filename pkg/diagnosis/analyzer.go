package diagnosis

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
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
