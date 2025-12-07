package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理配置文件",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "生成默认配置文件",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		configPath := filepath.Join(home, ".kubehealer.yaml")

		// 默认配置内容
		content := `
# KubeHealer 配置文件
monitor:
  namespace: "default"
  labels: ""
  interval: "5m"
`
		err := os.WriteFile(configPath, []byte(content), 0644)
		if err != nil {
			fmt.Printf("❌ 创建失败: %v\n", err)
			return
		}
		fmt.Printf("✅ 配置文件已生成: %s\n", configPath)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
}
