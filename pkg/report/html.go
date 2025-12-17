package report

import (
	"html/template"
	"os"
	"time"

	"github.com/swfoodt/kubehealer/pkg/diagnosis"
)

// HTMLData 传给模板的数据结构
// 把 DiagnosisResult 包装一下，加个“生成时间”字段
type HTMLData struct {
	diagnosis.DiagnosisResult
	GenerateTime string
}

// GenerateHTML 生成 HTML 文件
func GenerateHTML(result diagnosis.DiagnosisResult, filename string) error {
	// 1. 准备数据
	data := HTMLData{
		DiagnosisResult: result,
		GenerateTime:    time.Now().Format("2006-01-02 15:04:05"),
	}

	// 2. 解析模板 (从 templates.go 中的常量读取)
	tmpl, err := template.New("report").Parse(HTMLTemplate)
	if err != nil {
		return err
	}

	// 3. 创建文件
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// 4. 渲染
	return tmpl.Execute(f, data)
}
