package diagnosis

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake" // 关键：引入 fake 包
)

func TestAnalyzer_AnalyzePod(t *testing.T) {
	// 1. 准备假数据 (Mock Pod)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       "12345",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test-container",
					State: corev1.ContainerState{
						// 模拟 OOM
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "OOMKilled",
							ExitCode: 137,
						},
					},
				},
			},
		},
	}

	// 2. 创建假客户端 (Fake Clientset)
	// 这个 fakeClient 实现了 kubernetes.Interface，所以可以直接传给 NewAnalyzer
	fakeClient := fake.NewSimpleClientset(pod)

	// 3. 初始化分析器 (注入假客户端)
	analyzer := NewAnalyzer(fakeClient)

	// 4. 执行分析
	result := analyzer.AnalyzePod(pod)

	// 5. 验证结果
	if len(result.Containers) == 0 {
		t.Fatal("Expected container diagnosis, got none")
	}

	diag := result.Containers[0]
	foundOOM := false
	for _, issue := range diag.Issues {
		if issue.Title == "内存溢出 (OOMKilled)" {
			foundOOM = true
			break
		}
	}

	if !foundOOM {
		t.Error("AnalyzePod failed to detect OOMKilled via Mock client")
	}
}
