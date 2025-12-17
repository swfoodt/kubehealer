# KubeHealer 多平台构建脚本
# 功能: 自动注入版本信息，并生成 Windows/Linux/macOS 二进制文件
$content = Get-Content -Path ".\build.ps1" -Raw; [System.IO.File]::WriteAllText("$PWD\build.ps1", $content, [System.Text.Encoding]::UTF8)

$ErrorActionPreference = "Stop"

# 1. 准备构建目录
$OutputDir = "bin"
if (-not (Test-Path $OutputDir)) {
    # 修复: 这里之前写错了变量名，已修正为 $OutputDir
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
    Write-Host "[*] 创建目录: $OutputDir" -ForegroundColor Gray
}

# 2. 获取版本信息
try {
    $Version = git describe --tags --abbrev=0 2>$null
    if (-not $Version) { $Version = "v0.0.0" }
} catch {
    $Version = "v0.0.0"
}
$GitCommit = git rev-parse --short HEAD
$BuildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"

# 3. 定义目标平台列表
$Platforms = @(
    @{ OS = "windows"; Arch = "amd64"; Ext = ".exe" },
    @{ OS = "linux";   Arch = "amd64"; Ext = ""     },
    @{ OS = "darwin";  Arch = "amd64"; Ext = ""     }
)

Write-Host "🔨 开始构建 KubeHealer (Version: $Version)" -ForegroundColor Cyan
Write-Host "--------------------------------------------------"

# 4. 循环构建
foreach ($p in $Platforms) {
    $TargetOS = $p.OS
    $TargetArch = $p.Arch
    $Extension = $p.Ext
    
    $OutputName = "kubehealer-${TargetOS}-${TargetArch}${Extension}"
    $OutputPath = Join-Path $OutputDir $OutputName
    
    Write-Host "   -> 正在构建: $TargetOS / $TargetArch ..." -NoNewline

    # 设置环境变量
    $env:CGO_ENABLED = "0"
    $env:GOOS = $TargetOS
    $env:GOARCH = $TargetArch

    # 构造 LdFlags
    $LdFlags = "-s -w -X 'main.Version=$Version' -X 'main.GitCommit=$GitCommit' -X 'main.BuildTime=$BuildTime'"

    # 执行编译
    try {
        go build -ldflags $LdFlags -o $OutputPath ./cmd
        Write-Host " [成功]" -ForegroundColor Green
    } catch {
        Write-Host " [失败]" -ForegroundColor Red
        Write-Error $_
    }
}

# 5. 清理环境变量
$env:GOOS = $null
$env:GOARCH = $null

Write-Host "--------------------------------------------------"
Write-Host "✅ 所有构建完成！产物位于 ./$OutputDir 目录" -ForegroundColor Cyan
Get-ChildItem $OutputDir | Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize