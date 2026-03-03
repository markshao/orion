# DevSwarm - AI-Native Development Environment Manager

## 项目概要

**DevSwarm** 是一个专为 AI Coding 时代设计的开发环境管理工具。它通过提供一层抽象，将传统的 Git 分支虚拟化为多个并发的 **Node（执行单元）**，每个 Node 拥有独立的 Git Worktree 和 Tmux Session。

这种设计使得开发者（人类或 AI Agent）可以在同一逻辑分支上并发工作，互不干扰，完美支持 "Human-in-the-loop" 和 "Multi-Agent" 的协作模式。

## 核心概念

### 1. Logical Branch (逻辑分支)

用户认知中的常规 Git 分支，例如 `feature/login`。这是代码合并和版本控制的主线。

### 2. Node (执行单元)

DevSwarm 的核心并发单元。每个 Node 包含：

- **独立的文件系统隔离**：基于 Git Worktree，位于 `nodes/<node-name>/`。
- **独立的进程隔离**：基于 Tmux Session，Session 名与 Node 名绑定。
- **独立的版本控制隔离**：基于 Shadow Branch（物理分支）。

### 3. Shadow Branch (物理分支)

系统自动管理的 Git 分支，命名规则通常为 `devswarm/<logical-branch>/<node-name>`。它允许在不污染主逻辑分支的情况下进行实验性开发。

## 目录结构

DevSwarm 在项目根目录下维护以下结构：

```text
myapp_devswarm/
├── repo/                # 主仓库 (Bare Repo 或包含 .git 的主 Worktree)
├── nodes/               # 所有 Node 的 Worktree 挂载点
│   ├── login-test/      # Node: login-test
│   ├── login-review/    # Node: login-review
│   └── payment-dev/     # Node: payment-dev
└── .devswarm/           # 元数据与配置
    ├── state.json       # 全局状态 (Node 列表, Repo 信息)
    └── config.yaml      # 用户配置
```

## 当前实现状态 (Current Implementation)

项目采用 **Golang** 编写，使用 `cobra` 作为 CLI 框架。

### 核心模块

1.  **Workspace Manager (`internal/workspace`)**
    - 负责管理 `.devswarm` 目录和 `state.json`。
    - 维护 Node 的生命周期（创建、删除、查询）。
    - 持久化存储 Node 元数据 (Name, LogicalBranch, ShadowBranch, TmuxSession)。

2.  **Git Provider (`internal/git`)**
    - 封装 `os/exec` 调用 Git 命令。
    - 支持 `git clone`, `git worktree add/remove`, `git branch`.
    - 屏蔽了底层 Git 操作的复杂性。

3.  **Tmux Provider (`internal/tmux`)**
    - 封装 Tmux 会话管理。
    - 支持 `new-session` (后台创建) 和 `attach` (接管当前终端)。
    - 使用 `syscall.Exec` 确保 `enter` 命令能完美替换当前 Shell 进程。

### 已实现命令 (CLI)

| 命令                             | 描述                                   | 状态      |
| :------------------------------- | :------------------------------------- | :-------- |
| `devswarm init <url>`            | 初始化 DevSwarm 环境，克隆主仓库       | ✅ 已实现 |
| `devswarm spawn <branch> <name>` | 创建新 Node (Worktree + Shadow Branch) | ✅ 已实现 |
| `devswarm enter <name>`          | 进入 Node 对应的 Tmux Session          | ✅ 已实现 |
| `devswarm ls`                    | 列出所有活跃 Node 及其状态             | ✅ 已实现 |
| `devswarm rm <name>`             | 删除 Node (清理 Worktree 和 Session)   | ✅ 已实现 |
| `devswarm merge`                 | 将 Node 代码合并回逻辑分支             | 🚧 待开发 |

## 快速开始

### 1. 安装与构建

```bash
go build -o bin/devswarm main.go
export PATH=$PWD/bin:$PATH
```

### 2. 使用流程

```bash
# 1. 初始化项目
mkdir myproject_swarm && cd myproject_swarm
devswarm init https://github.com/user/repo.git

# 2. 创建一个用于测试功能的 Node
devswarm spawn feature/auth auth-test-node

# 3. 进入该 Node 的开发环境
devswarm enter auth-test-node
# (此时你位于 tmux session 中，目录已切换到 nodes/auth-test-node)

# 4. 退出 Tmux (Ctrl+b, d) 回到主控台

# 5. 查看状态
devswarm ls

# 6. 清理
devswarm rm auth-test-node
```

## 待办事项 (Roadmap)

- [ ] **Merge Command**: 实现 `devswarm merge`，支持将 Shadow Branch 的变更 Squash Merge 回 Logical Branch。
- [ ] **Agent Hooks**: 为 Node 生命周期添加 Hook (post-spawn, pre-merge)，以便自动启动 AI Agent。
- [ ] **Conflict Handling**: 增强 Git 操作的错误处理，当 Worktree 创建失败或分支冲突时提供更友好的提示。
- [ ] **Config Management**: 完善 `config.yaml` 支持，允许自定义 Tmux 配置文件路径等。
