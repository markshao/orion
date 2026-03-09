# Installation Guide

[English](installation.md) | [简体中文](installation_zh-CN.md)

## Prerequisites (环境要求)

Before installing Orion, ensure you have the following dependencies installed:

1.  **Git**: Version 2.20+ (Required for `git worktree` support).
2.  **Tmux**: Required for session management.
    ```bash
    # macOS
    brew install tmux
    ```
3.  **Qwen**: The AI agent runtime. Ensure `qwen` is in your PATH.
4.  **Go**: Version 1.21+ (Required to build from source).

## One-Click Install (Recommended)

You can install Orion using the following command:

```bash
curl -fsSL https://raw.githubusercontent.com/bytedance/Orion/main/install.sh | bash
```

## Manual Install (Build from Source)

If you prefer to build from source or contribute to the project:

```bash
# 1. Clone the repository
git clone https://github.com/bytedance/Orion.git
cd Orion

# 2. Build the binary
go build -o bin/orion main.go

# 3. Add to PATH (Optional)
export PATH=$PWD/bin:$PATH
```

## Verification (验证安装)

Run the following command to verify the installation:

```bash
orion --version
# Should output: Orion version v1.0.0
```

## Next Steps

Once installed, you can start by initializing a workspace. See [Human Node Guide](human-node.md).
