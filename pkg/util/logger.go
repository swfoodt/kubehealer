package util

import (
	"os"

	"github.com/sirupsen/logrus"
)

// InitLogger 初始化全局日志配置
func InitLogger(debug bool) {
	// 设置输出到标准输出
	logrus.SetOutput(os.Stdout)

	// 设置日志格式为文本格式 (带有颜色)
	// 如果在生产容器里，通常会设为 &logrus.JSONFormatter{}
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})

	// 设置日志级别
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("🔧 Debug 模式已开启")
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}
