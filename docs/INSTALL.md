# 📦 安装指南 (Installation Guide)

KubeHealer 支持多种安装方式，您可以根据实际环境选择最适合的一种。

## 方式一：下载预编译二进制文件 (推荐)

这是最简单的方式，无需安装 Go 环境，直接下载即可运行。

1. 访问 [GitHub Releases](https://github.com/swfoodt/kubehealer/releases) 页面。

2. 根据您的操作系统下载对应的文件：
   - **Windows**: `kubehealer-windows-amd64.exe`
   - **Linux**: `kubehealer-linux-amd64`
   - **macOS**: `kubehealer-darwin-amd64`

3. (Linux/macOS) 赋予执行权限：

```bash
   chmod +x kubehealer-linux-amd64
   mv kubehealer-linux-amd64 /usr/local/bin/kubehealer
```

4. 验证安装：

```Bash
kubehealer version
```


## 方式二：从源码编译安装

如果您想体验最新特性或参与开发，可以使用源码编译。

**前置要求**:

- Go 1.22+
- Git

**步骤**:

1. 克隆仓库：

```Bash 
git clone https://github.com/swfoodt/kubehealer.git
cd kubehealer
```

1. 编译：

- **Windows (PowerShell)**:

```PowerShell
.\build.ps1
```

- **Linux / macOS**:

```Bash
go build -o kubehealer ./cmd
```

编译产物将位于 `bin/` 目录下。