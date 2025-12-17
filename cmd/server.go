package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var port string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动 Web 服务器展示诊断报告",
	Long:  `启动一个轻量级 HTTP 服务器，托管 reports 目录，允许通过浏览器查看历史报告。`,
	Run: func(cmd *cobra.Command, args []string) {
		reportDir := "reports"

		// 确保目录存在
		if _, err := os.Stat(reportDir); os.IsNotExist(err) {
			_ = os.Mkdir(reportDir, 0755)
		}

		// 核心逻辑：将 reportDir 目录作为一个静态文件服务器
		fs := http.FileServer(http.Dir(reportDir))

		// 注册路由: "/"
		http.Handle("/", http.StripPrefix("/", fs))

		fmt.Printf("🌐 Web 服务器已启动: http://localhost:%s\n", port)
		fmt.Printf("📂 托管目录: ./%s\n", reportDir)
		fmt.Println("按 Ctrl+C 停止服务...")

		// 启动监听
		err := http.ListenAndServe(":"+port, nil)
		if err != nil {
			fmt.Printf("❌ 启动失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	// 支持自定义端口，默认为 8080
	serverCmd.Flags().StringVarP(&port, "port", "p", "8080", "Web 服务器监听端口")
}
