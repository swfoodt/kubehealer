# KubeHealer E2E 测试脚本 (v3 编码修复版)
# 功能: 自动部署故障 Pod -> 运行诊断 -> 验证关键词 -> 清理环境

$ErrorActionPreference = "Stop"

# 1. 强制控制台输出使用 UTF-8，防止显示乱码
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# 2. 编译项目
Write-Host "[*] [Step 1] 正在编译 KubeHealer..." -ForegroundColor Cyan
go build -o kubehealer.exe ./cmd
if (-not (Test-Path "kubehealer.exe")) {
    Write-Error "编译失败，未找到 kubehealer.exe"
}

# 定义测试用例
$testCases = @(
    @{
        Name = "CrashLoopBackOff 测试"
        Yaml = "test/manifests/crash-pod.yaml"
        PodName = "crash-pod"
        Expected = "ExitCode: 1" 
        WaitSec = 25 
    },
    @{
        Name = "ImagePullBackOff 测试"
        Yaml = "test/manifests/image-error-pod.yaml"
        PodName = "image-error-pod"
        Expected = "镜像拉取失败"
        WaitSec = 20
    },
    @{
        Name = "OOMKilled 测试"
        Yaml = "test/manifests/oom-pod.yaml"
        PodName = "oom-pod"
        Expected = "内存溢出"
        WaitSec = 15
    },
    @{
        Name = "Pending 测试"
        Yaml = "test/manifests/pending-pod.yaml"
        PodName = "pending-pod"
        # 你的截图显示 '无法调度' 没有出现在前500字，我们匹配更通用的特征
        # 或者增加错误日志打印长度。根据截图，PendingPod 应该会触发 PendingRule
        Expected = "无法调度" 
        WaitSec = 5
    }
)

# 统计结果
$passed = 0
$failed = 0

foreach ($case in $testCases) {
    Write-Host "\
--------------------------------------------------"
    Write-Host "[*] 正在执行: $($case.Name)" -ForegroundColor Yellow
    
    # 3. 清理旧环境
    $podName = $case.PodName
    kubectl delete pod $podName --ignore-not-found --wait=true 2>$null | Out-Null
    
    # 4. 部署故障 Pod
    Write-Host "   -> 部署 YAML: $($case.Yaml)"
    kubectl apply -f $case.Yaml | Out-Null
    
    # 5. 等待
    Write-Host "   -> 等待 $($case.WaitSec) 秒让故障复现..."
    Start-Sleep -Seconds $case.WaitSec
    
    # 6. 执行诊断
    Write-Host "   -> 执行诊断..."
    try {
        # 捕获所有输出
        $output = .\kubehealer.exe diagnose $podName 2>&1 | Out-String
    } catch {
        $output = $_.Exception.Message
    }

    # 7. 验证结果
    if ($output -match $case.Expected) {
        Write-Host "   [+] PASS: 成功检测到关键词 '$($case.Expected)'" -ForegroundColor Green
        $passed++
    } else {
        Write-Host "   [-] FAIL: 未检测到关键词 '$($case.Expected)'" -ForegroundColor Red
        Write-Host "   实际输出片段 (前1000字符):" 
        # 增加打印长度到 1000，防止诊断信息被截断
        Write-Host ($output.Substring(0, [Math]::Min($output.Length, 1000)))
        $failed++
    }

    # 8. 清理
    kubectl delete pod $podName --ignore-not-found --wait=false 2>$null | Out-Null
}

Write-Host "\
=================================================="
Write-Host "测试汇总: 通过: $passed  |  失败: $failed" -ForegroundColor Cyan

if ($failed -gt 0) {
    exit 1
} else {
    Write-Host "🎉 所有 E2E 测试通过！" -ForegroundColor Green
}