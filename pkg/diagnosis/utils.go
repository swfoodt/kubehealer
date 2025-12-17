package diagnosis

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ExplainExitCode 将数字退出码转换为人类可读的字符串
func ExplainExitCode(code int32) string {
	var exitCodeMap = map[int32]string{
		0:   "Completed (正常退出)",
		1:   "General Error (应用内部错误)",
		2:   "Misuse of Shell Builtins (Shell内建命令误用)",
		126: "Invoked Command Cannot Execute (命令不可执行)",
		127: "Command Not Found (命令未找到)",
		128: "Invalid Exit Argument (无效的退出参数)",
		130: "Script Terminated by Control-C (被Ctrl+C终止)",
		137: "SIGKILL (强制终止/OOMKilled - 内存溢出)",
		143: "SIGTERM (优雅终止)",
	}

	if msg, ok := exitCodeMap[code]; ok {
		return fmt.Sprintf("%d (%s)", code, msg)
	}

	if code > 128 {
		return fmt.Sprintf("%d (Signal %d)", code, code-128)
	}

	return fmt.Sprintf("%d (未知错误码)", code)
}

// SumRestarts 计算重启总数
func SumRestarts(pod *corev1.Pod) int32 {
	var count int32
	for _, cs := range pod.Status.ContainerStatuses {
		count += cs.RestartCount
	}
	return count
}

// TranslateTimestamp 翻译时间戳
func TranslateTimestamp(t time.Time) string {
	if t.IsZero() {
		return "未知"
	}
	duration := time.Since(t)
	if duration.Seconds() < 60 {
		return fmt.Sprintf("%.0f秒前", duration.Seconds())
	}
	if duration.Minutes() < 60 {
		return fmt.Sprintf("%.0f分钟前", duration.Minutes())
	}
	return fmt.Sprintf("%.0f小时前", duration.Hours())
}
