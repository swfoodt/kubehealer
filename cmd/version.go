package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// 这些变量在编译时通过 -ldflags 注入
// 如果直接运行 go run，它们会保持默认值
var (
	Version   = "dev"     // 版本号 (e.g., v1.0.0)
	GitCommit = "none"    // Git 哈希 (e.g., a1b2c3d)
	BuildTime = "unknown" // 构建时间
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🚑 KubeHealer Version Info:\n")
		fmt.Printf("   Version:    %s\n", Version)
		fmt.Printf("   Git Commit: %s\n", GitCommit)
		fmt.Printf("   Build Time: %s\n", BuildTime)
		fmt.Printf("   Go Version: %s\n", runtime.Version())
		fmt.Printf("   OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
