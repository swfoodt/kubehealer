package diagnosis

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Analyzer 负责编排整个 Pod 的诊断流程。
// 它依赖 RuleEngine 进行具体的规则匹配，并聚合所有诊断结果。
type Analyzer struct {
	client kubernetes.Interface
	engine *RuleEngine // 诊断引擎
}

// NewAnalyzer 初始化一个新的诊断分析器。
// client 参数应该是已连接的 K8s 客户端集合。
func NewAnalyzer(client kubernetes.Interface) *Analyzer {
	return &Analyzer{
		client: client,
		engine: NewRuleEngine(), // 初始化诊断引擎
	}
}

// AnalyzePod 对指定的 Pod 进行全方位的健康检查。
// 包括：基础状态、容器详情、事件流、日志分析以及规则引擎匹配。
func (a *Analyzer) AnalyzePod(pod *corev1.Pod) DiagnosisResult {
	result := DiagnosisResult{
		PodName:      pod.Name,
		Namespace:    pod.Namespace,
		NodeName:     pod.Spec.NodeName,
		Phase:        string(pod.Status.Phase),
		RestartCount: SumRestarts(pod),
		Containers:   []ContainerDiagnosis{},
		Events:       a.GetPodEvents(pod), // 获取事件列表
	}
	// 遍历容器进行诊断
	for _, cs := range pod.Status.ContainerStatuses {
		// 寻找对应的 Container Spec
		var targetContainer *corev1.Container
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == cs.Name {
				targetContainer = &pod.Spec.Containers[i]
				break
			}
		}

		// 获取单容器诊断结果
		containerDiag := a.GetContainerDiagnosis(pod, cs, targetContainer)
		result.Containers = append(result.Containers, containerDiag)
	}

	// 如果 Pod 是 Pending 且没有容器状态，手动触发一次诊断
	if len(pod.Status.ContainerStatuses) == 0 && pod.Status.Phase == corev1.PodPending {
		// 构造虚拟状态触发检查,构造一个空的 dummy 状态，为了触发 PendingRule
		dummyStatus := corev1.ContainerStatus{Name: "n/a"}
		containerDiag := a.GetContainerDiagnosis(pod, dummyStatus, nil)
		// 如果真的发现了问题（比如 PendingRule 命中了），才加进去
		if len(containerDiag.Issues) > 0 {
			result.Containers = append(result.Containers, containerDiag)
		}
	}

	return result
}

// GetContainerDiagnosis 返回 ContainerDiagnosis 结构体
func (a *Analyzer) GetContainerDiagnosis(pod *corev1.Pod, cs corev1.ContainerStatus, containerSpec *corev1.Container) ContainerDiagnosis {
	diag := ContainerDiagnosis{
		Name:   cs.Name,
		Ready:  cs.Ready,
		Issues: []Issue{},
	}

	// 填充基础状态信息
	if cs.State.Waiting != nil {
		diag.State = "Waiting"
		diag.Reason = cs.State.Waiting.Reason
		diag.Message = cs.State.Waiting.Message
	} else if cs.State.Terminated != nil {
		diag.State = "Terminated"
		diag.Reason = cs.State.Terminated.Reason
		diag.Message = cs.State.Terminated.Message
		diag.ExitCode = cs.State.Terminated.ExitCode
	} else if cs.State.Running != nil {
		diag.State = "Running"
	}

	// 填充资源配置
	if containerSpec != nil {
		diag.ResourceInfo = a.GetResourceInfo(*containerSpec)
	}

	// ----------------------------------------------------
	// 规则引擎介入
	// ----------------------------------------------------
	ruleResult := a.engine.Run(pod, containerSpec, cs)
	if ruleResult != nil {
		issueType := "Warning"
		if ruleResult.Title == "内存溢出 (OOMKilled)" {
			issueType = "Error"
		}

		diag.Issues = append(diag.Issues, Issue{
			Type:       issueType,
			Title:      ruleResult.Title,
			RawError:   ruleResult.RawError,
			Suggestion: ruleResult.Suggestion,
		})
	}

	// 检查 LastTerminationState (兜底补充)
	// 如果规则引擎没有覆盖这部分，可以在这里补充 Issue，或者完全依赖规则引擎。
	// 目前为了保持逻辑完整，我们还是加上这个检查，作为 Issue 添加进去。
	if cs.LastTerminationState.Terminated != nil {
		// last := cs.LastTerminationState.Terminated
		// 如果规则引擎还没报 OOM，这里补充信息
		// (简单起见，这里我们只把上次退出作为一条 Info 级别的 Issue 或者拼接在 Reason 里？)
		// 为了结构化，我们暂时不在这里加，因为 Day 13 的规则引擎应该已经处理了 OOM。
		// 如果只是普通退出，我们这里不需要额外处理，除非想展示历史。
	}

	// ----------------------------------------------------
	// 日志分析 (Day 27)
	// ----------------------------------------------------
	// 只有当容器不正常 (非 Running) 或者有重启记录时，才去抓日志
	// 避免抓取正常运行的日志浪费资源
	if cs.State.Running == nil || cs.RestartCount > 0 {
		logResult := AnalyzeContainerLogs(a.client, pod, cs.Name)
		diag.Logs = logResult.Logs
		diag.LogKeywords = logResult.MatchedKeyords

		// 如果日志里发现了严重错误，也可以生成一个 Issue
		if len(logResult.MatchedKeyords) > 0 {
			diag.Issues = append(diag.Issues, Issue{
				Type:       "Error",
				Title:      fmt.Sprintf("日志中发现错误特征: %s", strings.Join(logResult.MatchedKeyords, ", ")),
				Suggestion: "请查看下方详细日志定位代码问题",
			})
		}
	}

	return diag
}

// GetResourceInfo 格式化资源配置 (返回纯字符串)
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

	return fmt.Sprintf("CPU(Req=%s/Lim=%s) | Mem(Req=%s/Lim=%s)",
		reqCPU, limCPU, reqMem, limMem)
}

// GetPodEvents 返回字符串切片
func (a *Analyzer) GetPodEvents(pod *corev1.Pod) []string {
	var result []string

	// 使用 FieldSelector 过滤出涉及该 Pod 的事件
	// involvedObject.uid = Pod UID (更精确，防止同名冲突)
	selector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.uid=%s",
		pod.Name, pod.Namespace, pod.UID)

	events, err := a.client.CoreV1().Events(pod.Namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: selector,
	})

	if err != nil {
		return []string{fmt.Sprintf("❌ 获取事件失败: %v", err)}
	}

	if len(events.Items) == 0 {
		return []string{}
	}

	// 优化: 仅保留最近 1 小时的事件
	var recentEvents []corev1.Event
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	// 辅助函数: 获取事件发生的最佳时间 (解决 [未知] 问题)
	getEventTime := func(e corev1.Event) time.Time {
		if !e.LastTimestamp.IsZero() {
			return e.LastTimestamp.Time
		}
		if !e.EventTime.IsZero() {
			return e.EventTime.Time
		}
		// 如果都没有，尝试 FirstTimestamp
		if !e.FirstTimestamp.IsZero() {
			return e.FirstTimestamp.Time
		}
		return time.Time{} // 真的一无所有
	}

	for _, e := range events.Items {
		t := getEventTime(e)
		// 只要时间有效，且在1小时内，就保留
		if !t.IsZero() && t.After(oneHourAgo) {
			recentEvents = append(recentEvents, e)
		}
	}

	// 3. 按时间排序 (使用 recentEvents 而不是 events.Items)
	sort.Slice(recentEvents, func(i, j int) bool {
		t1 := getEventTime(recentEvents[i])
		t2 := getEventTime(recentEvents[j])
		return t1.Before(t2)
	})

	// 4. 截取最近 5 条
	start := 0
	if len(recentEvents) > 5 {
		start = len(recentEvents) - 5
	}

	for i := start; i < len(recentEvents); i++ {
		e := recentEvents[i] // ✅ 这里使用 recentEvents

		// 获取用于展示的时间
		t := getEventTime(e)
		age := TranslateTimestamp(t) // 确保 TranslateTimestamp 能处理 time.Time

		icon := "🔹"
		if e.Type == "Warning" {
			icon = "🔸"
		}
		result = append(result, fmt.Sprintf("%s [%s] %s: %s", icon, age, e.Reason, e.Message))
	}

	if len(result) == 0 {
		result = append(result, "暂无近期事件")
	}

	return result
}
