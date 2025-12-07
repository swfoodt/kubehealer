package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/swfoodt/kubehealer/pkg/diagnosis"
	"github.com/swfoodt/kubehealer/pkg/k8s"
	"github.com/swfoodt/kubehealer/pkg/report"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "实时监控 Pod 状态变化 (Informer模式)",
	Long:  `启动一个长运行进程，监听集群内 Pod 的创建、更新和删除事件。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 启动 KubeHealer 监控模式 (按 Ctrl+C 退出)...")

		// 1. 初始化客户端
		client, err := k8s.NewClient()
		if err != nil {
			fmt.Printf("❌ 连接失败: %v\n", err)
			os.Exit(1)
		}

		// 2. 创建 SharedInformerFactory
		// defaultResync: 0 表示不进行强制的全量同步（除非断连重连）
		factory := informers.NewSharedInformerFactory(client.Clientset, 0)

		// 3. 获取 Pod 的 Informer
		podInformer := factory.Core().V1().Pods().Informer()

		// 4. 注册事件处理函数 (Add, Update, Delete)
		podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// 当有新 Pod 创建时触发
				pod := obj.(*corev1.Pod)
				fmt.Printf("[➕ Added] %s/%s (Status: %s)\n", pod.Namespace, pod.Name, pod.Status.Phase)

				// 如果新 Pod 一上来就是 Pending (或者可能卡住了) 或者 Failed，诊断它
				// 这里只要不是 Running/Succeeded 就诊断一下
				if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
					go triggerDiagnosis(pod, client) // 使用 go 协程，不阻塞监控主线程
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				// 当 Pod 发生变化时触发 (这是最频繁的)
				oldPod := oldObj.(*corev1.Pod)
				newPod := newObj.(*corev1.Pod)

				// 为了避免刷屏，只有状态改变时才打印
				if oldPod.Status.Phase == newPod.Status.Phase &&
					oldPod.Status.ContainerStatuses[0].RestartCount == newPod.Status.ContainerStatuses[0].RestartCount {
					// 状态没变，重启次数没变，忽略（过滤掉单纯的 ResourceUpdate 等噪音）
					return
				}

				fmt.Printf("[🔄 Updated] %s/%s: %s -> %s (Restarts: %d)\n",
					newPod.Namespace, newPod.Name, oldPod.Status.Phase, newPod.Status.Phase,
					newPod.Status.ContainerStatuses[0].RestartCount)

				// 自动诊断逻辑
				// 1. 如果变成了非 Running 状态 (比如 Failed, Unknown)
				// 2. 或者虽然是 Running，但重启次数增加了 (CrashLoopBackOff 的特征)
				isCrashLoop := newPod.Status.ContainerStatuses[0].RestartCount > oldPod.Status.ContainerStatuses[0].RestartCount

				if newPod.Status.Phase != corev1.PodRunning || isCrashLoop {
					go triggerDiagnosis(newPod, client)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// 当 Pod 被删除时触发
				// 删除就不诊断了，人都没了
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
					if !ok {
						return
					}
					pod, ok = tombstone.Obj.(*corev1.Pod)
					if !ok {
						return
					}
				}
				fmt.Printf("[❌ Deleted] %s/%s\n", pod.Namespace, pod.Name)
			},
		})

		// 5. 启动 Informer
		// 使用 channel 来控制停止
		stopper := make(chan struct{})
		defer close(stopper)

		// 这是一个非阻塞调用，会在后台启动所有注册的 Informer
		factory.Start(stopper)

		// 6. 等待缓存同步 (重要！)
		// 必须等待它把集群里现有的 Pod 都拉取到本地缓存，才能认为是 Ready
		fmt.Println("⏳ 正在同步缓存...")
		if !cache.WaitForCacheSync(stopper, podInformer.HasSynced) {
			fmt.Println("❌ 缓存同步超时")
			return
		}
		fmt.Println("✅ 缓存同步完成，开始监听事件...")

		// 7. 阻塞主进程，直到收到 Ctrl+C
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\n👋 监控停止")
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}

// triggerDiagnosis 触发一次诊断并生成报告
// 需要传入 client 用于初始化 Analyzer
func triggerDiagnosis(pod *corev1.Pod, client *k8s.Client) {
	// 1. 初始化分析器
	analyzer := diagnosis.NewAnalyzer(client.Clientset)

	// 2. 执行诊断
	result := analyzer.AnalyzePod(pod)

	// 3. 只有当确实发现问题时（Containers里有Issue，或者Events里有Warn），才生成报告
	// 这里做一个简单的判断：如果 Phase 不是 Running/Succeeded，或者 RestartCount > 0
	// 生产环境可以做得更细，这里我们简单点：只要触发了就生成报告

	// 4. 生成 HTML 报告
	reportDir := "reports"
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		_ = os.Mkdir(reportDir, 0755)
	}

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_auto_%s.html", pod.Name, timestamp) // 加个 _auto_ 前缀区分
	fullPath := filepath.Join(reportDir, fileName)

	err := report.GenerateHTML(result, fullPath)
	if err != nil {
		fmt.Printf("❌ [%s] 报告生成失败: %v\n", pod.Name, err)
	} else {
		// 获取绝对路径方便点击
		absPath, _ := filepath.Abs(fullPath)
		fmt.Printf("🚨 [%s] 异常检测! 诊断报告已生成: %s\n", pod.Name, absPath)
	}
}
