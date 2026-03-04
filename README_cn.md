# DevSwarm 🐝

[English](README.md) | 中文

**DevSwarm** 是一个专为 AI Native 开发时代设计的开发环境管理工具，旨在促进人类开发者与 AI Agent 之间的高效并发协作。

在 AI Coding 时代，传统的 Git 工作流难以支持在同一个逻辑分支上进行多任务并发。DevSwarm 通过将 Git 分支虚拟化为 **Nodes（节点）** 来解决这个问题，每个节点都提供了一个隔离的环境（Worktree + Tmux Session），使得人类或 Agent 可以在互不干扰的情况下独立工作。

## 🚀 核心特性

- **节点抽象 (Node Abstraction)**：屏蔽了 `git worktree` 的复杂性，提供了一个清晰的接口来管理并发开发单元。
- **环境隔离 (Environment Isolation)**：每个节点拥有独立的文件系统 (Worktree) 和运行时会话 (Tmux)，确保零干扰。
- **影子分支 (Shadow Branching)**：自动管理 `影子分支` (例如 `devswarm/login-test/feature/login`)，允许在同一个逻辑功能上并行工作。
- **AI 就绪 (AI-Ready)**：作为 AI Agent 的基础设施设计，允许它们在自己的沙盒节点中生成、编码和测试。
- **状态管理 (State Management)**：持久化工作区状态，确保在重启或崩溃后能够恢复。

## 📦 安装

### 前置要求

请确保您的系统已安装以下工具：

- **Git** (v2.20+)
- **Tmux** (v3.0+)

### 通过脚本安装

我们提供了一个自动化安装脚本，可以从 GitHub 下载最新的二进制发布版本并配置自动补全。

```bash
curl -sL https://raw.githubusercontent.com/markshao/DevSwarm/main/install.sh | bash
```

该脚本将执行以下操作：

1. 检测您的操作系统和架构。
2. 从 [GitHub Releases](https://github.com/markshao/DevSwarm/releases) 下载最新的二进制文件。
3. 安装 `devswarm` 到 `/usr/local/bin`。
4. 为您的 Shell (Zsh/Bash) 配置自动补全。

## 🎮 演练场 (Playground)

DevSwarm 附带了一个自动化的 **端到端 (E2E) 测试脚本**，模拟完整的用户工作流。这是在不污染环境的情况下了解 DevSwarm 工作原理的最佳方式。

```bash
# 运行 E2E 测试脚本
./playground/test_e2e.sh
```

该脚本将：

- 初始化工作区。
- 创建 (Spawn) 一个节点。
- 模拟编码工作。
- 合并更改。
- 清理资源。

## 🛠 使用指南

### 1. 初始化工作区

为 Git 仓库创建一个 DevSwarm 工作区。

```bash
# 自动创建 'repo_workspace' 目录 (例如 DevSwarm_workspace)
devswarm init https://github.com/markshao/DevSwarm.git

# 或者指定自定义目录名称
devswarm init https://github.com/markshao/DevSwarm.git my_custom_workspace
```

> **注意**：所有后续命令必须在工作区目录内运行。

### 2. 创建节点 (Spawn)

为并发任务创建隔离节点。`devswarm` 支持两种模式：

#### 功能模式 (默认)
直接在功能分支上工作。最适合开发新功能。

```bash
# 直接在 'feature/login' 分支上创建节点。如果分支不存在，则从 'main' 创建。
devswarm spawn feature/login login-dev --base main --purpose coding
```

#### 影子模式 (Shadow Mode, --shadow)
基于目标分支创建一个临时的影子分支 (`ds-shadow/...`)。最适合代码审查、测试或实验性更改，而不会污染主分支。

```bash
# 基于 'feature/login' 创建一个审查节点，而不检出该分支本身
devswarm spawn feature/login login-review --shadow --purpose review
```

### 3. 进入节点 (Tmux)

进入节点的隔离开发环境 (Tmux Session)。

```bash
devswarm enter login-dev
```

_您现在位于 Tmux 会话中。工作目录是 `workspaces/login-dev`。使用 `Ctrl+b, d` 分离会话。_

### 4. 列出节点

检查所有活跃节点的状态。

```bash
devswarm ls
```

_输出:_

```text
NODE          BRANCH         PURPOSE      SESSION   CREATED
login-dev     feature/login  coding       RUNNING   2023-10-27T10:00:00Z
login-test    feature/login  testing      STOPPED   2023-10-27T10:05:00Z
```

### 5. 合并节点

将节点 (影子分支) 的更改合并回主逻辑分支。

```bash
# 压缩合并 (Squash merge) 并保留节点
devswarm merge login-dev

# 压缩合并并自动移除节点
devswarm merge login-dev --cleanup
```

### 6. 移除节点

手动移除节点并释放所有资源 (Worktree, Branch, Session)。

```bash
devswarm rm login-test
```

## 💻 IDE 集成 (Trae / VS Code)

DevSwarm 提供了与现代 IDE 的深度集成，以增强您的工作流。

### 1. 工作区集成 (推荐)

DevSwarm 会自动生成并维护一个标准的 `.code-workspace` 文件。导入此文件允许您在单个 IDE 窗口中同时编辑多个节点的文件，并获得完整的语言服务器支持。

**使用 Trae IDE:**

1. **导入工作区**:
   - 使用 **File** -> **Open Workspace...** 加载配置。
   - 选择根目录下生成的 `{project}.code-workspace` 文件。
   - Trae 将识别多根结构并索引所有节点。

2. **AI 上下文**:
   - 通过打开工作区，Trae 的 AI 可以访问所有活跃节点的上下文，允许跨节点重构和理解。

*(注: VS Code 用户也可以通过 **File** -> **Open Workspace from File...** 打开相同的 `.code-workspace` 文件)*

### 2. 终端集成

DevSwarm 可以根据您当前编辑的文件，自动将您的 IDE 终端连接到正确的 Node Tmux 会话。

**设置说明:**

1. 打开 IDE 的 `settings.json`。
2. 添加或替换以下配置：

```json
  "terminal.integrated.env.osx": {
    "CURRENT_FILE": "${file}"
  },
  "terminal.integrated.profiles.osx": {
    "devswarm-tmux": {
      "path": "/usr/local/bin/devswarm",
      "args": ["auto-attach", "${env:CURRENT_FILE}"],
      "icon": "terminal-tmux",
      "cwd": "${fileDirname}"
    }
  },
  "terminal.integrated.defaultProfile.osx": "devswarm-tmux"
```

> **注意**：我们使用 `CURRENT_FILE` 环境变量，因为 VS Code 的参数扩展 (`${file}`) 在某些上下文中可能表现不一致。

**工作原理:**

- 当您打开终端 (`Ctrl + ~`) 时，DevSwarm 会检测当前活动文件 (通过 `CURRENT_FILE` 传递) 是否属于特定的 **Node**。
- 如果是，您将自动连接到该节点的 **Tmux 会话**。
- 如果不是，您将连接到 **默认** 会话。

## 🏗 架构

- **主仓库 (Main Repo)**：唯一的真理来源 (Git 仓库)。
- **工作区 (Workspaces)**：从仓库派生的临时 Worktree。
- **状态 (State)**：`state.json` 文件跟踪节点、Tmux 会话和 Git 分支之间的映射关系。

```text
my_project_swarm/
├── .devswarm/
│   └── state.json
├── main_repo/           # 主 .git 仓库
├── workspaces/          # 活跃节点
│   ├── login-human/     # Worktree A
│   └── login-test/      # Worktree B
└── my_project.code-workspace # VSCode 工作区配置
```

## 📝 许可证

Apache-2.0 LICENSE
