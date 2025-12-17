package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/swfoodt/kubehealer/pkg/diagnosis"
	"github.com/swfoodt/kubehealer/pkg/k8s"
	"github.com/swfoodt/kubehealer/pkg/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// 定义过滤参数变量
var (
	monitorNamespace string
	monitorLabels    string
	monitorInterval  time.Duration
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "实时监控 Pod 状态变化 (Informer模式)",
	Long:  `启动一个长运行进程，监听集群内 Pod 的创建、更新和删除事件。支持通过 Namespace 和 Label 进行过滤。`,
	Run: func(cmd *cobra.Command, args []string) {
		// Day 28: 从 Viper 获取最终配置 (覆盖全局变量)
		// 如果命令行没传，就用配置文件的；如果传了，Viper 会自动用命令行的
		ns := viper.GetString("monitor.namespace")
		labels := viper.GetString("monitor.labels")
		interval := viper.GetDuration("monitor.interval")

		logrus.Info("🚀 启动 KubeHealer 监控模式(ctrl+c退出)...")
		logrus.Infof("   - 监听 Namespace: %s\n", ns)
		logrus.Infof("   - 监听 Labels: %s\n", labels)
		logrus.Infof("   - 同步间隔: %s\n", interval)

		// 初始化客户端
		client, err := k8s.NewClient()
		if err != nil {
			logrus.Errorf("❌ 连接失败: %v\n", err)
			os.Exit(1)
		}

		// 创建 SharedInformerFactory (带过滤选项)
		// 使用 WithOptions 支持 Namespace 和 LabelSelector
		var factory informers.SharedInformerFactory

		// 构造 ListOptions
		tweakListOptions := func(options *metav1.ListOptions) {
			if labels != "" {
				options.LabelSelector = labels
			}
		}

		if ns != "" {
			// 如果指定了 Namespace，只监听该 Namespace
			factory = informers.NewSharedInformerFactoryWithOptions(
				client.Clientset,
				interval,
				informers.WithNamespace(ns),
				informers.WithTweakListOptions(tweakListOptions),
			)
		} else {
			// 否则监听所有 Namespace
			factory = informers.NewSharedInformerFactoryWithOptions(
				client.Clientset,
				interval,
				informers.WithTweakListOptions(tweakListOptions),
			)
		}

		// 获取 Pod 的 Informer
		podInformer := factory.Core().V1().Pods().Informer()

		// 注册事件处理函数
		podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*corev1.Pod)
				logrus.Infof("[➕ Added] %s/%s (Status: %s)\n", pod.Namespace, pod.Name, pod.Status.Phase)

				if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
					go triggerDiagnosis(pod, client)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldPod := oldObj.(*corev1.Pod)
				newPod := newObj.(*corev1.Pod)

				// 【修复】安全地获取重启次数
				// 如果 Pod 处于 Pending 状态，ContainerStatuses 可能是空的，直接访问 [0] 会 panic
				var oldRestarts, newRestarts int32
				if len(oldPod.Status.ContainerStatuses) > 0 {
					oldRestarts = oldPod.Status.ContainerStatuses[0].RestartCount
				}
				if len(newPod.Status.ContainerStatuses) > 0 {
					newRestarts = newPod.Status.ContainerStatuses[0].RestartCount
				}

				// 只有状态发生实质变化才关心 (Phase 变了，或者重启次数变了)
				if oldPod.Status.Phase == newPod.Status.Phase && oldRestarts == newRestarts {
					// fmt.Println("Resync triggered for", newPod.Name) //测试-interval功能用
					return
				}

				logrus.Infof("[🔄 Updated] %s/%s: %s -> %s (Restarts: %d)\n",
					newPod.Namespace, newPod.Name, oldPod.Status.Phase, newPod.Status.Phase,
					newRestarts)

				// 自动诊断逻辑
				// 如果变成了非 Running 状态，或者重启次数增加了
				isCrashLoop := newRestarts > oldRestarts
				if newPod.Status.Phase != corev1.PodRunning || isCrashLoop {
					go triggerDiagnosis(newPod, client)
				}
			},
			DeleteFunc: func(obj interface{}) {
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
				logrus.Errorf("[❌ Deleted] %s/%s\n", pod.Namespace, pod.Name)
			},
		})

		// 启动
		stopper := make(chan struct{})
		defer close(stopper)
		factory.Start(stopper)

		logrus.Info("⏳ 正在同步缓存...")
		if !cache.WaitForCacheSync(stopper, podInformer.HasSynced) {
			logrus.Error("❌ 缓存同步超时")
			return
		}
		logrus.Info("✅ 开始监听...")

		// 优雅退出
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logrus.Info("\n👋 收到停止信号，正在退出...")
	},
}

// 去重缓存 (PodUID -> 上次诊断时间)
// 使用 sync.Map 保证并发安全
var diagnosisCooldown sync.Map

// triggerDiagnosis 触发一次诊断并生成报告
func triggerDiagnosis(pod *corev1.Pod, client *k8s.Client) {
	// 去重检查
	// 冷却时间设置为 1 分钟
	const cooldownPeriod = 1 * time.Minute

	// 获取上次诊断时间
	if lastTime, loaded := diagnosisCooldown.Load(pod.UID); loaded {
		if time.Since(lastTime.(time.Time)) < cooldownPeriod {
			// 如果还在冷却期内，直接跳过
			logrus.Infof("⏳ [%s] 处于冷却期，跳过重复诊断\n", pod.Name)
			return
		}
	}

	// 记录本次诊断时间 (相当于更新缓存)
	diagnosisCooldown.Store(pod.UID, time.Now())

	// 初始化分析器 (以下逻辑保持不变)
	analyzer := diagnosis.NewAnalyzer(client.Clientset)
	result := analyzer.AnalyzePod(pod)

	// 生成报告
	reportDir := "reports"
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		_ = os.Mkdir(reportDir, 0755)
	}

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_auto_%s.html", pod.Name, timestamp)
	fullPath := filepath.Join(reportDir, fileName)

	err := report.GenerateHTML(result, fullPath)
	if err != nil {
		logrus.Errorf("❌ [%s] 报告生成失败: %v\n", pod.Name, err)
	} else {
		absPath, _ := filepath.Abs(fullPath)
		logrus.Infof("🚨 [%s] 异常检测! 诊断报告已生成: %s\n", pod.Name, absPath)
	}
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	// 注册 Flags
	monitorCmd.Flags().StringVarP(&monitorNamespace, "namespace", "n", "", "指定监控的 Namespace (默认为所有)")
	monitorCmd.Flags().StringVarP(&monitorLabels, "label-selector", "l", "", "指定监控的 Label Selector (例如: app=nginx)")
	// 默认 10 分钟同步一次，避免长时间运行导致缓存漂移
	monitorCmd.Flags().DurationVarP(&monitorInterval, "interval", "i", 10*time.Minute, "Informer 全量同步时间间隔 (例如 10m, 1h)")

	// 2. 绑定 Viper (让 Viper 知道这些 Flag 的存在)
	viper.BindPFlag("monitor.namespace", monitorCmd.Flags().Lookup("namespace"))
	viper.BindPFlag("monitor.labels", monitorCmd.Flags().Lookup("label-selector"))
	viper.BindPFlag("monitor.interval", monitorCmd.Flags().Lookup("interval"))
}
