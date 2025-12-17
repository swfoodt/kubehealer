package diagnosis

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// -----------------------------------------------------------
// OOMRule: 检测内存溢出
// -----------------------------------------------------------
type OOMRule struct{}

func (r *OOMRule) Name() string {
	return "OOMRule"
}

func (r *OOMRule) Check(pod *corev1.Pod, container *corev1.Container, status corev1.ContainerStatus) CheckResult {
	// 无论是 Waiting 还是 Terminated，都要检查 LastTerminationState

	var termState *corev1.ContainerStateTerminated

	// 检查当前是否 Terminated
	if status.State.Terminated != nil {
		termState = status.State.Terminated
	}
	// 检查上次是否 Terminated (LastTerminationState)
	if status.LastTerminationState.Terminated != nil {
		// 优先取最新的状态，但如果当前是 Waiting，我们就看上一次
		if termState == nil {
			termState = status.LastTerminationState.Terminated
		}
	}

	// 如果根本没有终止记录，直接跳过
	if termState == nil {
		return CheckResult{Matched: false}
	}

	// 核心判断: Reason == "OOMKilled"
	if termState.Reason == "OOMKilled" {
		res := CheckResult{
			Matched:  true,
			Title:    "内存溢出 (OOMKilled)",
			RawError: fmt.Sprintf("Exit Code: %s", ExplainExitCode(termState.ExitCode)),
		}

		// 资源建议
		if container != nil {
			limit := container.Resources.Limits.Memory()
			if !limit.IsZero() {
				res.Suggestion = fmt.Sprintf("检测到内存限制 Limit=%s，建议适当调大", limit.String())
			} else {
				res.Suggestion = "未设置内存限制，建议设置 Limits 防止节点资源耗尽"
			}
		}
		return res
	}

	return CheckResult{Matched: false}
}

// -----------------------------------------------------------
// ImagePullRule: 检测镜像拉取失败
// -----------------------------------------------------------
type ImagePullRule struct{}

func (r *ImagePullRule) Name() string {
	return "ImagePullRule"
}

func (r *ImagePullRule) Check(pod *corev1.Pod, container *corev1.Container, status corev1.ContainerStatus) CheckResult {
	// 只关心 Waiting 状态
	if status.State.Waiting == nil {
		return CheckResult{Matched: false}
	}

	reason := status.State.Waiting.Reason
	if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
		return CheckResult{
			Matched:    true,
			Title:      fmt.Sprintf("镜像拉取失败 (无法获取 %s)", status.Image),
			RawError:   status.State.Waiting.Message,
			Suggestion: "请检查: 1.镜像名拼写 2.镜像Tag是否存在 3.私有仓库ImagePullSecrets权限",
		}
	}

	return CheckResult{Matched: false}
}

// -----------------------------------------------------------
// CrashRule: 检测容器反复重启 (CrashLoopBackOff)
// -----------------------------------------------------------
type CrashRule struct{}

func (r *CrashRule) Name() string {
	return "CrashRule"
}

func (r *CrashRule) Check(pod *corev1.Pod, container *corev1.Container, status corev1.ContainerStatus) CheckResult {
	// 如果是 Waiting 且原因是 CrashLoopBackOff
	if status.State.Waiting != nil && status.State.Waiting.Reason == "CrashLoopBackOff" {
		res := CheckResult{
			Matched:    true,
			Title:      "容器反复重启 (CrashLoopBackOff)",
			RawError:   status.State.Waiting.Message,
			Suggestion: "应用程序启动失败，请检查应用日志 (logs) 或配置",
		}

		// 尝试从 LastTerminationState 获取更多信息
		if status.LastTerminationState.Terminated != nil {
			last := status.LastTerminationState.Terminated
			res.RawError += fmt.Sprintf(" | 上次退出: %s (%s)",
				ExplainExitCode(last.ExitCode), last.Reason)
		}

		return res
	}
	return CheckResult{Matched: false}
}
