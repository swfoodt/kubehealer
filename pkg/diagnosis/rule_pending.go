package diagnosis

import (
	corev1 "k8s.io/api/core/v1"
)

// -----------------------------------------------------------
// PendingRule: 检测调度失败
// -----------------------------------------------------------
type PendingRule struct{}

func (r *PendingRule) Name() string {
	return "PendingRule"
}

func (r *PendingRule) Check(pod *corev1.Pod, container *corev1.Container, status corev1.ContainerStatus) CheckResult {
	// Pending 状态下，Pod 可能还没有 ContainerStatus
	// 但我们的架构是遍历 ContainerStatus 调用的。
	// 如果 Pod 处于 Pending，ContainerStatus 里的 State 通常是 Waiting (ContainerCreating) 或者根本没数据。

	// 这里我们需要一种机制：如果是 Pending，且 Reason 是 Unschedulable

	// 1. 检查 Pod 整体状态
	if pod.Status.Phase == corev1.PodPending {
		// 检查 Pod Conditions 里的 PodScheduled 字段
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
				return CheckResult{
					Matched:    true,
					Title:      "Pod 无法调度 (Pending)",
					RawError:   cond.Message,
					Suggestion: "集群资源不足或不满足调度策略 (NodeSelector/Taint)，请查看下方 Events 详情",
				}
			}
		}
	}

	return CheckResult{Matched: false}
}
