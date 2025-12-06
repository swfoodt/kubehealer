package diagnosis

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// Analyzer 负责具体的诊断逻辑
type Analyzer struct {
	// 未来可以在这里添加配置或缓存
}

// NewAnalyzer 创建一个新的分析器
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// AnalyzePod 执行 Pod 的全面诊断并打印结果
func (a *Analyzer) AnalyzePod(pod *corev1.Pod) {
	// 1. 打印基础信息
	fmt.Printf("📦 Pod: %s | 命名空间: %s | 节点: %s\n",
		pod.Name, pod.Namespace, pod.Spec.NodeName)

	fmt.Printf("   状态: %s | 重启总数: %d\n",
		pod.Status.Phase, sumRestarts(pod))

	// 2. 打印容器状态详情 (这里是 Day 5 逻辑的升级版)
	fmt.Println("   --- 容器详情 ---")
	for _, cs := range pod.Status.ContainerStatuses {
		a.analyzeContainerStatus(cs)
	}
}

// 辅助函数: 计算所有容器的重启次数总和
func sumRestarts(pod *corev1.Pod) int32 {
	var count int32
	for _, cs := range pod.Status.ContainerStatuses {
		count += cs.RestartCount
	}
	return count
}

// analyzeContainerStatus 分析单个容器的状态
func (a *Analyzer) analyzeContainerStatus(cs corev1.ContainerStatus) {
	prefix := fmt.Sprintf("   ├─ 容器: %s", cs.Name)

	// Case 1: Waiting (例如 CrashLoopBackOff, ImagePullBackOff)
	if cs.State.Waiting != nil {
		fmt.Printf("%s\n", prefix)
		fmt.Printf("   └─ ⚠️  状态: Waiting | 原因: %s | 信息: %s\n",
			cs.State.Waiting.Reason, cs.State.Waiting.Message)
		return
	}

	// Case 2: Terminated (例如 Error, OOMKilled)
	if cs.State.Terminated != nil {
		fmt.Printf("%s\n", prefix)
		fmt.Printf("   └─ 🛑 状态: Terminated | 原因: %s | 退出码: %d | 信息: %s\n",
			cs.State.Terminated.Reason, cs.State.Terminated.ExitCode, cs.State.Terminated.Message)
		return
	}

	// Case 3: Running
	if cs.State.Running != nil {
		// 如果虽然 Running 但有重启过，也标记一下
		if cs.RestartCount > 0 {
			fmt.Printf("%s\n", prefix)
			fmt.Printf("   └─ ⚠️  状态: Running (但已重启 %d 次)\n", cs.RestartCount)
		} else {
			fmt.Printf("%s\n", prefix)
			fmt.Printf("   └─ ✅ 状态: Running\n")
		}
		return
	}
}
