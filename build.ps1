# KubeHealer 构建脚本
# 功能: 自动获取 Git 信息并编译注入
$content = Get-Content -Path ".\build.ps1" -Raw; [System.IO.File]::WriteAllText("$PWD\build.ps1", $content, [System.Text.Encoding]::UTF8)
$ErrorActionPreference = "Stop"

# 1. 获取版本信息
# 尝试获取最新的 tag，如果没有 tag 则默认为 v0.0.0
try {
    $Version = git describe --tags --abbrev=0 2>$null
    if (-not $Version) { $Version = "v0.0.0" }
} catch {
    $Version = "v0.0.0"
}

# 获取当前的 Commit Hash
$GitCommit = git rev-parse --short HEAD

# 获取当前时间
$BuildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"

Write-Host "🔨 正在构建 KubeHealer..." -ForegroundColor Cyan
Write-Host "   Version:    $Version"
Write-Host "   Commit:     $GitCommit"
Write-Host "   BuildTime:  $BuildTime"

# 2. 构造 ldflags 参数
# 注意: PowerShell 中传递带引号的参数给外部命令需要特别小心
$LdFlags = "-s -w -X 'main.Version=$Version' -X 'main.GitCommit=$GitCommit' -X 'main.BuildTime=$BuildTime'"

# 3. 执行编译
# -s -w 可以减小二进制体积 (去掉调试符号)
go build -ldflags $LdFlags -o kubehealer.exe ./cmd

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ 构建成功: .\kubehealer.exe" -ForegroundColor Green
    # 验证一下
    .\kubehealer.exe version
} else {
    Write-Error "构建失败"
}