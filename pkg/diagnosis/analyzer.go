package diagnosis

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Analyzer struct {
	client *kubernetes.Clientset
	engine *RuleEngine // 诊断引擎
}

func NewAnalyzer(client *kubernetes.Clientset) *Analyzer {
	return &Analyzer{
		client: client,
		engine: NewRuleEngine(), // 初始化诊断引擎
	}
}

// AnalyzePod 编排诊断流程
func (a *Analyzer) AnalyzePod(pod *corev1.Pod) {
	// 获取并打印基础信息
	info := a.GetPodBasicInfo(pod)
	fmt.Println(info)

	// 获取并打印容器状态
	fmt.Println("   --- 容器详情 ---")

	// 如果 Pod 是 Pending 且没有容器状态，手动触发一次诊断
	if len(pod.Status.ContainerStatuses) == 0 {
		// 构造一个空的 dummy 状态，只为了触发 PendingRule
		dummyStatus := corev1.ContainerStatus{Name: "n/a"}
		msg := a.GetContainerStatus(pod, dummyStatus, nil)
		fmt.Println(msg)
	} else {
		// 正常遍历
		for _, cs := range pod.Status.ContainerStatuses {
			// 寻找对应的 Container Spec 以获取资源配置
			var targetContainer *corev1.Container
			for i := range pod.Spec.Containers {
				if pod.Spec.Containers[i].Name == cs.Name {
					targetContainer = &pod.Spec.Containers[i]
					break
				}
			}

			// 传入 pod 对象,获取单容器诊断结果
			statusMsg := a.GetContainerStatus(pod, cs, targetContainer)
			fmt.Println(statusMsg)
		}
	}
}

// GetPodBasicInfo 提取基础信息字符串
func (a *Analyzer) GetPodBasicInfo(pod *corev1.Pod) string {
	return fmt.Sprintf("📦 Pod: %s | 命名空间: %s | 节点: %s\n   状态: %s | 重启总数: %d",
		pod.Name, pod.Namespace, pod.Spec.NodeName,
		pod.Status.Phase, SumRestarts(pod))
}

// GetContainerStatus 解析单个容器状态
func (a *Analyzer) GetContainerStatus(pod *corev1.Pod, cs corev1.ContainerStatus, containerSpec *corev1.Container) string {
	prefix := fmt.Sprintf("   ├─ 容器: %s", cs.Name)

	// 只要能找到 Spec，就先把资源信息准备好
	var resourceInfo string
	if containerSpec != nil {
		resourceInfo = "\n" + a.GetResourceInfo(*containerSpec)
	}

	// ----------------------------------------------------
	// 规则引擎介入
	// ----------------------------------------------------
	result := a.engine.Run(pod, containerSpec, cs)
	if result != nil {
		// 如果规则引擎发现了问题，直接用规则引擎的结果
		icon := "⚠️ "
		// 如果是比较严重的错误，换个图标
		if result.Title == "内存溢出 (OOMKilled)" {
			icon = "🛑 "
		}

		output := fmt.Sprintf("%s\n   └─ %s %s", prefix, icon, result.Title)
		if result.RawError != "" {
			output += fmt.Sprintf(" | %s", result.RawError)
		}
		if result.Suggestion != "" {
			output += fmt.Sprintf("\n      💡 建议: %s", result.Suggestion)
		}
		return output + resourceInfo
	}

	// 如果规则引擎没发现问题 (Matched=false)，回退到原来的默认展示逻辑,保持旧逻辑作为 fallback
	// 1. Waiting
	if cs.State.Waiting != nil {
		return fmt.Sprintf("%s\n   └─ ⚠️  状态: Waiting | 原因: %s | 信息: %s",
			prefix, cs.State.Waiting.Reason, cs.State.Waiting.Message) + resourceInfo
	}

	// 2. Terminated
	if cs.State.Terminated != nil {
		return fmt.Sprintf("%s\n   └─ 🛑 状态: Terminated | 原因: %s | 退出码: %d",
			prefix, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode) + resourceInfo
	}

	// 3. Running
	status := fmt.Sprintf("%s\n   └─ ✅ 状态: Running", prefix)
	if cs.RestartCount > 0 {
		status += fmt.Sprintf(" (但已重启 %d 次)", cs.RestartCount)
	}
	return status + resourceInfo
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
		age := TranslateTimestamp(e.LastTimestamp.Time)

		icon := "🔹"
		if e.Type == "Warning" {
			icon = "🔸"
		}

		fmt.Printf("   %s [%s] %s: %s\n", icon, age, e.Reason, e.Message)
	}
}

// GetResourceInfo 格式化容器的资源配置
func (a *Analyzer) GetResourceInfo(container corev1.Container) string {
	req := container.Resources.Requests
	lim := container.Resources.Limits

	reqCPU := req.Cpu().String()
	reqMem := req.Memory().String()
	limCPU := lim.Cpu().String()
	limMem := lim.Memory().String()

	// 处理未设置的情况 (0)
	if reqCPU == "0" {
		reqCPU = "未设置"
	}
	if reqMem == "0" {
		reqMem = "未设置"
	}
	if limCPU == "0" {
		limCPU = "未设置"
	}
	if limMem == "0" {
		limMem = "未设置"
	}

	return fmt.Sprintf("      📊 资源配置: CPU(Req=%s/Lim=%s) | Mem(Req=%s/Lim=%s)",
		reqCPU, limCPU, reqMem, limMem)
}
