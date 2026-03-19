# <img src="assets/icon.svg" alt="Orion Logo" width="40" height="40" align="top"/> Orion: AI 原生开发环境管理器

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

**Orion** 是专为 **Agentic DevOps** 时代设计的 CLI 工具。它虚拟化了你的本地开发环境，让你能够与 AI Agents 像队友一样协同工作。

## 🌌 为什么叫 "Orion"?

**Orion (猎户座)** 是 AI Agents 的导航系统。这个名字象征着 **Orion 如何在你的代码库中编排 Agents**，像星座指引航海者一样，引导它们完成复杂的开发任务。

---

## 🌟 核心理念：Agentic DevOps

传统的 DevOps 依赖于远程 CI/CD 流水线——这通常是缓慢、无状态且与你的 IDE 断连的。

**Orion 将流水线带回了本地。** 它引入了 **Node (节点)** 的概念：
*   **Human Node (人类节点)**: 你的专属工作区 (Git Worktree + Tmux Session)。
*   **Agentic Node (智能体节点)**: 一个临时的后台工作区，AI Agents 可以在其中与你 *并发* 地编写代码、运行测试和修复 Bug。

### Local Agentic DevOps

Local Agentic DevOps 的核心是：把 CI/流水线能力带到你的本地机器上，让 AI Agents 以“本地队友”的方式参与研发闭环：

- **Human Node（人类节点）**：你在隔离的 worktree + tmux session 里正常开发与 commit。
- **Local Pipeline（本地流水线）**：commit 触发 workflow pipeline。
- **Agentic Nodes（智能体节点）**：流水线里的每个 step 都会启动一个独立的 worktree + tmux session，并在对应的 shadow branch 上执行任务。
- **Apply Loop（回归闭环）**：workflow 成功后，通过 `orion apply` 将 workflow 的结果分支合并回 Human Node 的分支。

### 本地流水线 (Release Workflow 示例)
 
 Orion 允许你通过配置 `.orion/workflows/*.yaml` 来定义 Agents 如何与你的代码库交互。一个强大的内置示例是 **发版工作流 (Release Workflow)**：
 
 当你的特性开发完毕准备发版时，可以执行：
 ```bash
 orion workflow run release-workflow <node-name>
 ```
 
 在底层，Orion 会将工作流解析为多个有序步骤执行：
 1. **Agent 步骤 (`rebase`)**：Orion 临时创建一个隔离的 `Agentic Node`（基于影子分支的独立 Worktree）。AI Agent 在这里把你的特性分支变基 (Rebase) 到 `main` 分支，自动解决潜在的代码冲突，运行测试并提交结果。
 2. **Bash 步骤 (`commit-check`)**：执行一个轻量级的脚本进行兜底，确保工作流中的 git 状态干净且可被合并。
 3. **Bash 步骤 (`merge`)**：自动将处理好冲突的影子分支合并 (Merge) 回你的 `Human Node`。
 
 这种机制让你能够安全地把解决冲突等繁琐任务抛给后台的 Agents，且**绝对不会污染你当前的主工作区**。
 
 <img src="assets/diagrams/local-agentic-devops.png" alt="Local Agentic DevOps diagram" width="900" />

---

## 🚀 快速开始

### 安装

**一键安装 (推荐)**

```bash
curl -fsSL https://raw.githubusercontent.com/markshao/orion/main/install.sh | bash
```

**手动安装**

如需源码构建，详见 [安装指南](user-guide/installation_zh-CN.md)。

### 使用

#### 1. 初始化
```bash
mkdir myproject_swarm && cd myproject_swarm
orion init https://github.com/user/repo.git
```

#### 2. 开始编码 (Human Node)

**方式 A: 手动创建**

```bash
# 为你的特性创建一个节点
orion spawn feature/login login-dev

# 进入隔离环境
orion enter login-dev
```

**方式 B: AI 智能创建 (推荐)**

使用自然语言让 AI 自动生成分支名和节点名：

```bash
# 使用自然语言创建节点
orion ai "实现用户登录功能"

# 基于特定分支开发
orion ai "基于 release/v1.2 修复支付 bug"

# 使用 --force 跳过确认
orion ai "重构认证模块" --force
```

`orion ai` 命令会：
- 使用 LLM (Moonshot/OpenAI 兼容) 分析你的描述
- 自动生成合适的分支名（如 `feature/user-login`、`fix/payment-bug`）
- 自动生成易读的节点名（如 `user-login-dev`、`payment-fix`）
- 创建 worktree 并搭建开发环境

**配置**

创建 `~/.orion.conf` 配置你的 AI 提供商：

```yaml
api_key: "$MOONSHOT_API_KEY"          # 或直接填写: "sk-xxx"
base_url: "https://api.moonshot.cn/v1"
model: "kimi-k2-turbo-preview"
```

然后进入开发环境：

```bash
orion enter <生成的节点名>
```

#### 3. Agent 协作
当你在 `login-dev` 中提交代码时，工作流会自动开始。

```bash
# 查看 Agent 状态
orion workflow ls

# 检查 Agent 做了什么
orion workflow inspect <run-id>

# 将 Agent 的更改变更回你的节点
orion apply login-dev
```

---

## ✨ 自动补全

安装脚本会尝试自动配置 Zsh 和 Bash 的自动补全。
如果需要手动设置：

**Zsh**
```bash
echo "source <(orion completion zsh)" >> ~/.zshrc
source ~/.zshrc
```

**Bash**
```bash
echo "source <(orion completion bash)" >> ~/.bashrc
source ~/.bashrc
```

## 📚 文档

- [**安装指南**](user-guide/installation_zh-CN.md): 环境要求与设置。
- [**Human Node 指南**](user-guide/human-node_zh-CN.md): 管理工作区与 VSCode 集成。
- [**Agentic Workflow 指南**](user-guide/workflow_zh-CN.md): 配置 Agent、触发器与回归闭环。

---

## 🛠 技术栈

- **Golang**: 核心逻辑与 CLI (Cobra)。
- **Git Worktree**: 文件系统隔离。
- **Tmux**: 进程与会话隔离。
- **Qwen**: 驱动自动化的 AI 引擎。

## License

Apache License 2.0
