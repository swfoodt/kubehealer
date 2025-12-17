package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/swfoodt/kubehealer/pkg/util"
)

// 全局变量
var (
	cfgFile string // 配置文件路径
	debug   bool   // 是否开启调试模式
)

// rootCmd 代表没有调用子命令时的基础命令
var rootCmd = &cobra.Command{
	Use:   "kubehealer",
	Short: "KubeHealer: K8s 容器层诊断工具",
	Long:  `一个基于 Go + client-go 的 Kubernetes 诊断工具，用于快速定位 Pod 异常。`,
	//在此处可以添加全局 flag，例如 --kubeconfig
}

// Execute 将所有子命令添加到根命令并设置标志
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局参数: --config
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件 (默认为 $HOME/.kubehealer.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "开启调试模式 (显示详细日志)")
}

// initConfig 读取配置文件和环境变量
func initConfig() {
	if cfgFile != "" {
		// 使用参数指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 查找 home 目录
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// 搜索路径: home 目录
		viper.AddConfigPath(home)
		// 搜索文件名: .kubehealer (无需后缀)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kubehealer")
	}

	// 读取环境变量 (自动将 KUBEHEALER_NAMESPACE 映射为 namespace)
	viper.SetEnvPrefix("KUBEHEALER")
	viper.AutomaticEnv()

	// 尝试读取配置
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("⚙️  已加载配置文件:", viper.ConfigFileUsed())
	}

	// 在配置读取完后，初始化日志
	util.InitLogger(debug)

	if err := viper.ReadInConfig(); err == nil {
		// 使用 logrus 打印，而不是 fmt
		// logrus.Infof("⚙️ 已加载配置文件: %s", viper.ConfigFileUsed())
		// 注意：这里可能还没法直接用 logrus，因为 import 循环问题
		// 暂时在 initConfig 里还是可以用 fmt 或者直接调 logrus
	}
}
