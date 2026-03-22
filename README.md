# <img src="assets/icon.svg" alt="Orion Logo" width="40" height="40" align="top"/> Orion: AI-Native Development Environment Manager

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

**Orion** is a CLI tool for the **Agentic DevOps** era. It virtualizes your local development environment so you can develop in isolated nodes and hand off integration work to AI agents running in parallel worktrees.

## 🌌 Why "Orion"?

**Orion** is a navigation system for AI agents. The name symbolizes how **Orion orchestrates agents across your codebase**, guiding them through complex development tasks like the constellation guides a navigator.

---

## 🌟 Core Concept: Agentic DevOps

Traditional DevOps relies on remote CI/CD pipelines—slow, stateless, and disconnected from your IDE.

**Orion brings the pipeline to your local machine.** It introduces the concept of **Nodes**:

- **Human Node**: Your dedicated workspace (Git Worktree + Tmux Session).
- **Agentic Node**: An ephemeral workspace where AI Agents running in the background can write code, run tests, and fix bugs _concurrently_ with you.

### Local Agentic DevOps

Local Agentic DevOps means **bringing CI-like integration work onto your local machine**, with AI agents running as first-class teammates:

- **Human Node**: you create a branch and node, code in an isolated worktree + tmux session, and commit normally.
- **Release Workflow**: when your feature is ready, you run `release-workflow` on that node.
- **Agentic Nodes**: workflow steps run in isolated worktrees + tmux sessions on shadow branches.
- **Ready to Push**: when the workflow succeeds, Orion marks the human node as `READY_TO_PUSH`.
- **Push**: you publish the validated branch with `orion push`.

### The Recommended Flow

Orion allows you to configure `.orion/workflows/*.yaml` to define how agents interact with your codebase. The recommended built-in path is the **Release Workflow**:

1. Use `orion ai` to generate a branch name and node name from a natural-language task.
2. Enter the human node and develop normally.
3. Commit your code on the human node.
4. Run `orion workflow run release-workflow --node <node-name>`.
5. Orion creates an isolated agentic node on a shadow branch.
6. The agent rebases onto the target base branch, resolves conflicts, and prepares the branch for integration.
7. If the workflow succeeds, the human node status becomes `READY_TO_PUSH`.
8. Run `orion push <node-name>` to publish the branch.

This lets you offload rebasing and conflict resolution to agents without polluting your main working directory.

<img src="assets/diagrams/local-agentic-devops.png" alt="Local Agentic DevOps diagram" width="900" />

1.  **Create**: Use AI to create a branch and human node.
2.  **Code**: Work and commit in your Human Node.
3.  **Delegate**: Run a workflow to let Agent Nodes handle rebase and conflict resolution.
4.  **Publish**: Push the validated result once the node becomes `READY_TO_PUSH`.

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
orion init https://github.com/user/repo.git
```

#### 2. Create Your Human Node

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

#### 3. Develop and Commit

After entering the node, develop normally and commit your changes:

```bash
orion enter login-dev
git status
git add .
git commit -m "feat: implement login flow"
```

#### 4. Run the Release Workflow

When your human-node changes are ready, run the built-in release workflow:

```bash
# Trigger the release workflow on a specific node
orion workflow run release-workflow --node login-dev

# Check workflow status
orion workflow ls

# Inspect what the agent did
orion workflow inspect <run-id>
```

The `release-workflow` uses an agentic node on a shadow branch to help with rebase and conflict handling. When the run succeeds, Orion marks the human node as `READY_TO_PUSH`.

#### 5. Push the Result

```bash
orion inspect login-dev
orion push login-dev
```

#### 6. Customize Workflows

You are not limited to the built-in release workflow. Orion can be extended through:

- `.orion/workflows/*.yaml` for workflow definitions
- `.orion/agents/*.yaml` for agent runtime configuration
- `.orion/prompts/*` for agent prompts and task instructions

That lets you define your own agentic nodes and automation steps beyond rebasing and conflict resolution.

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
- [**Agentic Workflow Guide**](user-guide/workflow.md): Configuring workflows, agents, and release automation.

---

## 🛠 Tech Stack

- **Golang**: Core logic and CLI (Cobra).
- **Git Worktree**: File system isolation.
- **Tmux**: Process and session isolation.
- **Qwen**: The AI engine powering the automation.

## License

Apache License 2.0
