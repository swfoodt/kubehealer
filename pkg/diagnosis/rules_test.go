package diagnosis

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOOMRule_Check(t *testing.T) {
	rule := &OOMRule{}

	// 定义测试用例
	tests := []struct {
		name        string
		status      corev1.ContainerStatus
		shouldMatch bool
	}{
		{
			name: "Case 1: 当前状态就是 OOMKilled",
			status: corev1.ContainerStatus{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						Reason:   "OOMKilled",
						ExitCode: 137,
					},
				},
			},
			shouldMatch: true,
		},
		{
			name: "Case 2: 当前是 CrashLoop，但上次死因是 OOMKilled (隐蔽的 OOM)",
			status: corev1.ContainerStatus{
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
				},
				LastTerminationState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						Reason:   "OOMKilled", // 关键点：历史死因
						ExitCode: 137,
					},
				},
			},
			shouldMatch: true,
		},
		{
			name: "Case 3: 正常运行",
			status: corev1.ContainerStatus{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{StartedAt: metav1.Now()},
				},
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 一个空的 Pod 和 Container (OOMRule 主要看 status)
			pod := &corev1.Pod{}
			container := &corev1.Container{}

			res := rule.Check(pod, container, tt.status)

			if res.Matched != tt.shouldMatch {
				t.Errorf("Check() matched = %v, want %v", res.Matched, tt.shouldMatch)
			}

			// 如果匹配了，检查标题是否正确
			if res.Matched && res.Title != "内存溢出 (OOMKilled)" {
				t.Errorf("Check() title = %v, want '内存溢出 (OOMKilled)'", res.Title)
			}
		})
	}
}
