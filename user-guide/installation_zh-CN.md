# 安装指南

[English](installation.md) | [简体中文](installation_zh-CN.md)

## 环境要求

在安装 Orion 之前，请确保已安装以下依赖：

1.  **Git**: 版本 2.20+ (需要支持 `git worktree`)。
2.  **Tmux**: 用于会话管理。
    ```bash
    # macOS
    brew install tmux
    ```
3.  **Qwen**: AI Agent 运行时。确保 `qwen` 在你的 PATH 中。
4.  **Go**: 版本 1.21+ (源码构建需要)。

## 一键安装 (推荐)

使用以下命令快速安装 Orion：

```bash
curl -fsSL https://raw.githubusercontent.com/bytedance/Orion/main/install.sh | bash
```

## 手动安装 (源码构建)

如果你想从源码构建或参与项目开发：

```bash
# 1. 克隆仓库
git clone https://github.com/bytedance/Orion.git
cd Orion

# 2. 编译二进制文件
go build -o bin/orion main.go

# 3. 添加到 PATH (可选)
export PATH=$PWD/bin:$PATH
```

## 验证安装

运行以下命令验证安装是否成功：

```bash
orion --version
# 应输出: Orion version v1.0.0
```

## 下一步

安装完成后，你可以开始初始化工作区。详见 [Human Node 指南](human-node_zh-CN.md)。
