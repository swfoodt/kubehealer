package diagnosis

import (
	corev1 "k8s.io/api/core/v1"
)

// RuleEngine 管理并执行所有注册的规则
type RuleEngine struct {
	rules []Rule
}

// NewRuleEngine 初始化引擎并加载默认规则
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: []Rule{
			&OOMRule{},       // 注册 OOM 规则
			&ImagePullRule{}, // 注册镜像拉取失败规则
			&CrashRule{},     // 注册崩溃循环规则
			&PendingRule{},   // 注册调度失败规则
		},
	}
}

// Register 添加新规则
func (e *RuleEngine) Register(r Rule) {
	e.rules = append(e.rules, r)
}

// Run 对单个容器运行所有规则，返回第一个命中的结果 (或者收集所有结果)
// 这里我们采取“短路”策略：一旦发现严重问题(Matched=true)，就返回
func (e *RuleEngine) Run(pod *corev1.Pod, container *corev1.Container, status corev1.ContainerStatus) *CheckResult {
	for _, rule := range e.rules {
		res := rule.Check(pod, container, status)
		if res.Matched {
			// 命中规则，返回结果
			return &res
		}
	}
	return nil // 没有命中任何异常规则
}
