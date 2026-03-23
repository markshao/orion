# <img src="assets/icon.svg" alt="Orion Logo" width="40" height="40" align="top"/> Orion: AI 原生开发环境管理器

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

**Orion** 是面向 **Agentic DevOps** 时代的 CLI 工具。它将本地开发环境虚拟化，让你可以在隔离节点中开发，并把集成类工作交给并行运行的 AI Agents。

## 🌌 为什么叫 "Orion"?

**Orion (猎户座)** 是 AI Agents 的导航系统。这个名字象征着 **Orion 如何在你的代码库中编排 Agents**，像星座指引航海者一样，引导它们完成复杂的开发任务。

---

## 🌟 核心理念：Agentic DevOps

传统的 DevOps 依赖于远程 CI/CD 流水线——这通常是缓慢、无状态且与你的 IDE 断连的。

**Orion 将流水线带回了本地。** 它引入了 **Node (节点)** 的概念：
*   **Human Node (人类节点)**: 你的专属工作区 (Git Worktree + Tmux Session)。
*   **Agentic Node (智能体节点)**: 一个临时的后台工作区，AI Agents 可以在其中与你 *并发* 地编写代码、运行测试和修复 Bug。

### Local Agentic DevOps

Local Agentic DevOps 的核心是：把 CI/集成类工作带回本地，让 AI Agents 以“本地队友”的方式参与研发闭环：

- **Human Node（人类节点）**：你先创建 branch 和 node，再在隔离的 worktree + tmux session 里正常开发与 commit。
- **Release Workflow（发版工作流）**：当特性准备就绪后，你在该节点上运行 `release-workflow`。
- **Agentic Nodes（智能体节点）**：workflow 里的 step 会启动独立的 worktree + tmux session，并在 shadow branch 上执行任务。
- **Ready to Push（可推送状态）**：workflow 成功后，Orion 会把 human node 标记为 `READY_TO_PUSH`。
- **Push（发布）**：最后通过 `orion push` 推送结果分支。

### 推荐开发流程
 
 Orion 允许你通过配置 `.orion/workflows/*.yaml` 来定义 Agents 如何与你的代码库交互。当前推荐的内置主路径是 **发版工作流 (Release Workflow)**：

 1. 使用 `orion ai` 根据自然语言生成 branch 名和 node 名。
 2. 进入 human node，正常开发。
 3. 在 human node 上提交代码。
 4. 运行 `orion workflow run release-workflow --node <node-name>`。
 5. Orion 创建隔离的 agentic node，并基于 shadow branch 执行 workflow。
 6. Agent 负责 rebase、处理冲突，并准备好可集成结果。
 7. 如果 workflow 成功，human node 状态会变成 `READY_TO_PUSH`。
 8. 运行 `orion push <node-name>` 推送分支。
 
 这种机制让你能够安全地把 rebase、冲突处理等繁琐任务抛给后台的 Agents，且**不会污染你当前的主工作区**。
 
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
orion init https://github.com/user/repo.git
```

#### 2. 创建 Human Node

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

#### 3. 开发并提交

进入节点后，按正常开发流程提交代码：

```bash
orion enter login-dev
git status
git add .
git commit -m "feat: implement login flow"
```

#### 4. 运行 Release Workflow

当 human node 上的代码准备好之后，运行内置的 release workflow：

```bash
# 在指定节点上触发 release workflow
orion workflow run release-workflow --node login-dev

# 查看 workflow 状态
orion workflow ls

# 检查 Agent 做了什么
orion workflow inspect <run-id>
```

`release-workflow` 会在 shadow branch 上启动 agentic node，帮助你完成 rebase 和冲突处理。workflow 成功后，Orion 会把 human node 标记为 `READY_TO_PUSH`。

#### 5. 推送结果

```bash
orion inspect login-dev
orion push login-dev
```

#### 6. 自定义 Workflows

你并不局限于内置的 release workflow。Orion 支持通过以下方式扩展：

- `.orion/workflows/*.yaml`：定义 workflow
- `.orion/agents/*.yaml`：配置 agent 运行时
- `.orion/prompts/*`：定义 agent prompt 和任务说明

这意味着你可以围绕自己的研发流程，扩展出更多 agentic nodes 和自动化步骤。

#### 7. Bare Repo Git 操作

Orion 会把 Git 数据存放在 `repo.git/`，把可编辑代码放在各个 node worktree 中。不带 `-w` 的 `orion run` 只用于 Git 操作，比如 fetch、打 tag、推 tag：

```bash
orion run git fetch origin
orion run git tag v1.0.0
orion run git push --tags
```

凡是需要工作树的命令，都应使用 `orion run -w <node>`。

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
- [**Agentic Workflow 指南**](user-guide/workflow_zh-CN.md): 配置 workflows、agents 与发版自动化。
- [**Orion Skills**](orion-skills/README_zh-CN.md): Orion 随仓库分发的可复用 agent skills，以及面向 Codex 的安装示例。

---

## 🛠 技术栈

- **Golang**: 核心逻辑与 CLI (Cobra)。
- **Git Worktree**: 文件系统隔离。
- **Tmux**: 进程与会话隔离。
- **Qwen**: 驱动自动化的 AI 引擎。

## License

Apache License 2.0
