# Installation Guide

[English](installation.md) | [简体中文](installation_zh-CN.md)

## Prerequisites (环境要求)

Before installing DevSwarm, ensure you have the following dependencies installed:

1.  **Git**: Version 2.20+ (Required for `git worktree` support).
2.  **Tmux**: Required for session management.
    ```bash
    # macOS
    brew install tmux
    ```
3.  **Trae Agent**: The AI agent runtime. Ensure `trae-agent` is in your PATH.
4.  **Go**: Version 1.21+ (Required to build from source).

## Building from Source (源码构建)

```bash
# 1. Clone the repository
git clone https://github.com/bytedance/DevSwarm.git
cd DevSwarm

# 2. Build the binary
go build -o bin/devswarm main.go

# 3. Add to PATH (Optional)
export PATH=$PWD/bin:$PATH
```

## Verification (验证安装)

Run the following command to verify the installation:

```bash
devswarm --version
# Should output: DevSwarm version v1.0.0
```

## Next Steps

Once installed, you can start by initializing a workspace. See [Human Node Guide](human-node.md).
