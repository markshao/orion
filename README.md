# <img src="assets/icon.svg" alt="Orion Logo" width="40" height="40" align="top"/> Orion: AI-Native Development Environment Manager

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

**Orion** is a CLI tool designed for the **Agentic DevOps** era. It virtualizes your local development environment, allowing you to collaborate with AI Agents as if they were teammates sitting next to you.

## 🌌 Why "Orion"?

**Orion** is a navigation system for AI agents. The name symbolizes how **Orion orchestrates agents across your codebase**, guiding them through complex development tasks like the constellation guides a navigator.

---

## 🌟 Core Concept: Agentic DevOps

Traditional DevOps relies on remote CI/CD pipelines—slow, stateless, and disconnected from your IDE.

**Orion brings the pipeline to your local machine.** It introduces the concept of **Nodes**:

- **Human Node**: Your dedicated workspace (Git Worktree + Tmux Session).
- **Agentic Node**: An ephemeral workspace where AI Agents running in the background can write code, run tests, and fix bugs _concurrently_ with you.

### Local Agentic DevOps

Local Agentic DevOps means **bringing CI-like automation onto your local machine**, with AI agents running as first-class teammates:

- **Human Node**: you code in an isolated worktree + tmux session and commit normally.
- **Local Pipeline**: the commit triggers a workflow pipeline.
- **Agentic Nodes**: each pipeline step runs in its own isolated worktree + tmux session on a shadow branch.
- **Apply Loop**: when the workflow is successful, you apply/merge the workflow result back onto your Human Node branch.

### The Local Pipeline (Release Workflow Example)

Orion allows you to configure `.orion/workflows/*.yaml` to define how agents interact with your codebase. A powerful built-in example is the **Release Workflow**:

When you are ready to ship your feature, you can trigger:
```bash
orion workflow run release-workflow <node-name>
```

Under the hood, Orion parses the workflow into steps:
1. **Agent Step (`rebase`)**: Orion creates an isolated `Agentic Node` (with its own Git Worktree on a Shadow Branch). The AI Agent rebases your feature branch onto `main`, automatically resolves conflicts, runs tests, and commits the result.
2. **Bash Step (`commit-check`)**: A lightweight script verifies the git status to ensure the shadow branch is in a clean, committable state.
3. **Bash Step (`merge`)**: Automatically fast-forwards or merges the resolved shadow branch back into your `Human Node`.

This mechanism allows you to offload tedious tasks (like conflict resolution) to agents safely in the background, without ever polluting your main working directory.

<img src="assets/diagrams/local-agentic-devops.png" alt="Local Agentic DevOps diagram" width="900" />

1.  **You Code**: Work in your Human Node.
2.  **Agents React**: On every commit, Orion spins up Agent Nodes.
3.  **Parallel Execution**: While you continue coding, Agent 1 writes tests, Agent 2 reviews code.
4.  **Loop Closed**: You use `orion apply` to merge the Agents' work back into your branch when you are ready.

---

## 🚀 Quick Start

### Installation

**One-Click Install (Recommended)**

```bash
curl -fsSL https://raw.githubusercontent.com/markshao/orion/main/install.sh | bash
```

**Manual Install**

See [Installation Guide](user-guide/installation.md) for building from source.

### Usage

#### 1. Initialize

```bash
mkdir myproject_swarm && cd myproject_swarm
orion init https://github.com/user/repo.git
```

#### 2. Start Coding (Human Node)

**Option A: Manual Creation**

```bash
# Create a node for your feature
orion spawn feature/login login-dev

# Enter the isolated environment
orion enter login-dev
```

**Option B: AI-Powered Creation (Recommended)**

Use natural language to let AI generate branch and node names for you:

```bash
# Create a node using natural language
orion ai "implement user login feature"

# Based on a specific branch
orion ai "fix payment bug based on release/v1.2"

# Skip confirmation with --force flag
orion ai "refactor authentication module" --force
```

The `orion ai` command will:
- Analyze your description using LLM (Moonshot/OpenAI-compatible)
- Auto-generate appropriate branch name (e.g., `feature/user-login`, `fix/payment-bug`)
- Auto-generate readable node name (e.g., `user-login-dev`, `payment-fix`)
- Create the worktree and set up the development environment

**Configuration**

Create `~/.orion.conf` to configure your AI provider:

```yaml
api_key: "$MOONSHOT_API_KEY"          # Or direct key: "sk-xxx"
base_url: "https://api.moonshot.cn/v1"
model: "kimi-k2-turbo-preview"
```

Then enter your development environment:

```bash
orion enter <generated-node-name>
```

#### 3. Agent Collaboration

When you commit code in `login-dev`, a workflow starts automatically.

```bash
# Check agent status
orion workflow ls

# Inspect what the agent did
orion workflow inspect <run-id>

# Merge agent's changes back to your node
orion apply login-dev
```

---

## ✨ Autocompletion

The install script attempts to configure autocompletion for Zsh and Bash automatically.
If you need to set it up manually:

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

## 📚 Documentation

- [**Installation Guide**](user-guide/installation.md): Requirements and setup.
- [**Human Node Guide**](user-guide/human-node.md): Managing your workspace and VSCode integration.
- [**Agentic Workflow Guide**](user-guide/workflow.md): Configuring agents, triggers, and the apply loop.

---

## 🛠 Tech Stack

- **Golang**: Core logic and CLI (Cobra).
- **Git Worktree**: File system isolation.
- **Tmux**: Process and session isolation.
- **Qwen**: The AI engine powering the automation.

## License

Apache License 2.0
