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

// -----------------------------------------------------------
// 诊断结果数据结构
// -----------------------------------------------------------

// DiagnosisResult 包含一个 Pod 的完整诊断信息
type DiagnosisResult struct {
	PodName      string               `json:"pod_name"`
	Namespace    string               `json:"namespace"`
	NodeName     string               `json:"node_name"`
	Phase        string               `json:"phase"`
	RestartCount int32                `json:"restart_count"`
	Containers   []ContainerDiagnosis `json:"containers"` // 容器级诊断列表
	Events       []string             `json:"events"`     // 最近的事件列表
}

// ContainerDiagnosis 单个容器的诊断详情
type ContainerDiagnosis struct {
	Name         string   `json:"name"`
	State        string   `json:"state"`         // Waiting, Running, Terminated
	Reason       string   `json:"reason"`        // CrashLoopBackOff, OOMKilled ...
	Message      string   `json:"message"`       // 详细信息
	ExitCode     int32    `json:"exit_code"`     // 退出码
	Ready        bool     `json:"ready"`         // 是否就绪
	ResourceInfo string   `json:"resource_info"` // CPU/Mem 配置字符串
	Issues       []Issue  `json:"issues"`        // 发现的问题 (由规则引擎产出)
	Logs         []string `json:"logs"`          // 抓取的最后几行日志
	LogKeywords  []string `json:"log_keywords"`  // 从日志中提取的关键词
}

// Issue 代表发现的一个具体问题
type Issue struct {
	Type       string `json:"type"`       // Error (🛑) or Warning (⚠️)
	Title      string `json:"title"`      // 标题
	RawError   string `json:"raw_error"`  // 原始报错
	Suggestion string `json:"suggestion"` // 修复建议
}
