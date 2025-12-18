# 📖 使用手册 (User Manual)

本文档将指导您如何使用 KubeHealer 解决实际生产环境中的 Kubernetes 故障。

## 1. 核心命令概览

| 命令 | 说明 | 示例 |
| :--- | :--- | :--- |
| `diagnose` | 诊断单个 Pod，分析根因 | `kubehealer diagnose pod-name` |
| `monitor` | 启动守护进程，实时监控并报警 | `kubehealer monitor -n default` |
| `server` | 启动 Web 界面查看历史报告 | `kubehealer server -p 8080` |
| `config` | 管理配置文件 | `kubehealer config init` |

## 2. 实战场景演示

### 场景 A：应用反复重启 (CrashLoopBackOff)

**现象**: 业务 Pod 状态不断在 Running 和 CrashLoopBackOff 之间切换。

**诊断**:
```bash
kubehealer diagnose payment-service-v3
````

**输出分析**: KubeHealer 会自动捕获容器最近一次的 **Exit Code** 和 **最后 50 行日志**。

- 如果发现 `panic: runtime error`，说明是代码 Bug。
    
- 如果发现 `Exit Code 137` 且没有日志，说明可能是 OOM（内存溢出）。
    

### 场景 B：Pod 一直处于 Pending 状态

**现象**: 部署新服务后，Pod 长时间卡在 Pending，不调度到节点上。

**诊断**:

Bash

```
kubehealer diagnose big-data-job-01
```

**输出分析**: 工具会分析 Pod 的 Request 资源与集群节点的 Allocatable 资源。

- 报告提示: `Insufficient cpu` -> 说明集群 CPU 资源不足。
    
- 报告提示: `node(s) had taint {node-role.kubernetes.io/master: }` -> 说明 Pod 缺少对应的容忍度 (Toleration)。
    

### 场景 C：镜像拉取失败 (ImagePullBackOff)

**现象**: Pod 状态为 ImagePullBackOff 或 ErrImagePull。

**诊断**:

Bash

```
kubehealer diagnose nginx-typo
```

**输出分析**: 工具会检查镜像名称格式及 Secret 权限。

- 建议: "请检查镜像 Tag 是否存在，或检查 ImagePullSecrets 是否配置正确。"
    

## 3. 高级功能：24小时监控模式

在生产环境中，我们不可能盯着屏幕看。您可以启动 Monitor 模式，让 KubeHealer 自动巡检。

Bash

```
# 启动监控，每 5 分钟全量同步一次，只监控 app=nginx 的 Pod
kubehealer monitor --namespace default --label-selector "app=nginx" --interval 5m
```

**效果**: 当 Pod 发生异常（如重启次数增加）时，KubeHealer 会：

1. 自动触发诊断。
    
2. 生成 HTML 报告保存到 `./reports` 目录。
    

## 4. 常见问题 (FAQ)

**Q: 执行 diagnose 时提示 "connection refused" 或 "dial tcp ... connect: ex"?**
A: 这通常意味着工具无法连接到 Kubernetes 集群。
1. 请检查您的 Kubernetes 集群是否正在运行 (例如 Minikube 是否启动)。
2. 尝试运行 `kubectl get pod`，如果 `kubectl` 也报错，说明是集群连接配置 (`~/.kube/config`) 的问题，而非 KubeHealer 的问题。

**Q: 报告中的中文乱码?** A: 请确保您的终端支持 UTF-8 编码。Windows 用户建议使用 Windows Terminal 或 PowerShell Core。