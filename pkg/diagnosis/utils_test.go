package diagnosis

import (
	"testing"
	"time"
)

// 测试 ExplainExitCode 函数
func TestExplainExitCode(t *testing.T) {
	tests := []struct {
		code     int32
		expected string
	}{
		// 修改期望值，加上 "数字 (...) " 的格式
		{0, "0 (Completed (正常退出))"},
		{137, "137 (SIGKILL (强制终止/OOMKilled - 内存溢出))"},
		// 修改测试用例：用一个小于 128 且不在 map 里的数来测“未知错误”
		{50, "50 (未知错误码)"},
		// 999 会命中 Signal 逻辑
		{999, "999 (Signal 871)"},
	}

	for _, tt := range tests {
		got := ExplainExitCode(tt.code)
		if got != tt.expected {
			t.Errorf("ExplainExitCode(%d) = %v; want %v", tt.code, got, tt.expected)
		}
	}
}

// 测试 TranslateTimestamp 函数
func TestTranslateTimestamp(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string // 注意：这里只能模糊匹配，或者固定 Mock 时间
	}{
		{"Zero Time", time.Time{}, "未知"},
		{"Just now", now.Add(-5 * time.Second), "5秒前"},
		{"One hour ago", now.Add(-2 * time.Hour), "2小时前"},
	}

	for _, tt := range tests {
		got := TranslateTimestamp(tt.input)
		if got != tt.expected {
			t.Errorf("TranslateTimestamp(%s) = %v; want %v", tt.name, got, tt.expected)
		}
	}
}
