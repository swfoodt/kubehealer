package diagnosis

import (
	"bufio"
	"context"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// 常见错误模式库
// 使用正则表达式匹配各种语言/框架的典型报错
var errorPatterns = map[string]*regexp.Regexp{
	"Java Exception":    regexp.MustCompile(`(?i)(Exception|Error):`),
	"Go Panic":          regexp.MustCompile(`panic:`),
	"Python Traceback":  regexp.MustCompile(`Traceback \(most recent call last\):`),
	"Node Error":        regexp.MustCompile(`(?i)ReferenceError|TypeError|SyntaxError`),
	"OOM Message":       regexp.MustCompile(`(?i)Kill process|Out of memory`),
	"Permission Denied": regexp.MustCompile(`(?i)permission denied`),
	"Common Error":      regexp.MustCompile(`(?i)(error|fail|fatal|exception)`),
}

// LogAnalysisResult 日志分析结果
type LogAnalysisResult struct {
	Logs           []string // 抓取的最后几行日志
	MatchedKeyords []string // 匹配到的错误关键字
}

// AnalyzeContainerLogs 获取并分析容器日志
func AnalyzeContainerLogs(client kubernetes.Interface, pod *corev1.Pod, containerName string) LogAnalysisResult {
	result := LogAnalysisResult{
		Logs:           []string{},
		MatchedKeyords: []string{},
	}

	// 配置日志获取选项
	lineLimit := int64(50) // 只看最后 50 行
	opts := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &lineLimit,
		// 如果容器当前挂了，尝试获取上一次运行的日志 (Previous)
		// 这对于 CrashLoopBackOff 特别重要！
		Previous: isContainerRestarted(pod, containerName),
	}

	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, opts)
	stream, err := req.Stream(context.TODO())
	if err != nil {
		// 如果获取失败（比如没有 Previous 日志），尝试获取当前日志
		if opts.Previous {
			opts.Previous = false
			req = client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, opts)
			stream, err = req.Stream(context.TODO())
		}

		if err != nil {
			result.Logs = append(result.Logs, fmt.Sprintf("❌ 无法获取日志: %v", err))
			return result
		}
	}
	defer stream.Close()

	// 扫描日志
	scanner := bufio.NewScanner(stream)
	uniqueMatches := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		result.Logs = append(result.Logs, line)

		// 正则匹配
		for name, pattern := range errorPatterns {
			if pattern.MatchString(line) {
				if !uniqueMatches[name] {
					uniqueMatches[name] = true
					result.MatchedKeyords = append(result.MatchedKeyords, name)
				}
			}
		}
	}

	return result
}

// 辅助函数：判断容器是否重启过（决定是否加 Previous 参数）
func isContainerRestarted(pod *corev1.Pod, containerName string) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == containerName {
			return cs.RestartCount > 0
		}
	}
	return false
}
