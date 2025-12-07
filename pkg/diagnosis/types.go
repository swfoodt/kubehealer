package diagnosis

import (
	corev1 "k8s.io/api/core/v1"
)

// CheckResult 代表单条规则的检查结果
type CheckResult struct {
	Matched    bool   // 是否命中了这条规则
	Title      string // 简短的标题 (例如 "内存溢出")
	Suggestion string // 修复建议 (例如 "建议增加 Limit")
	RawError   string // 原始报错信息
}

// Rule 是所有诊断规则必须实现的接口
type Rule interface {
	// Name 返回规则的唯一标识符
	Name() string

	// Check 执行检查
	// 参数: pod (整个Pod对象), container (当前容器Spec), status (当前容器状态)
	Check(pod *corev1.Pod, container *corev1.Container, status corev1.ContainerStatus) CheckResult
}
