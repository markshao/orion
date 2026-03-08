# DevSwarm: AI-Native Development Environment Manager

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

**DevSwarm** is a CLI tool designed for the **Agentic DevOps** era. It virtualizes your local development environment, allowing you to collaborate with AI Agents as if they were teammates sitting next to you.

---

## 🌟 Core Concept: Agentic DevOps

Traditional DevOps relies on remote CI/CD pipelines—slow, stateless, and disconnected from your IDE. 

**DevSwarm brings the pipeline to your local machine.** It introduces the concept of **Nodes**:
*   **Human Node**: Your dedicated workspace (Git Worktree + Tmux Session).
*   **Agentic Node**: An ephemeral workspace where AI Agents running in the background can write code, run tests, and fix bugs *concurrently* with you.

### The "Chain of Branch" Workflow

Instead of blocking your work, DevSwarm orchestrates a chain of **Shadow Branches**:

```mermaid
graph TD
    User((User)) -->|1. Commit| HumanNode["Human Node<br/>(feature/login)"]
    HumanNode -->|2. Trigger| Workflow{Workflow Engine}
    
    subgraph "Agentic Workflow (Local)"
        Workflow -->|3. Spawn| AgentNode1["Agent: Unit Test<br/>(shadow/ut)"]
        AgentNode1 -- Fixes & Commits --> AgentNode2["Agent: Code Review<br/>(shadow/cr)"]
    end
    
    AgentNode2 -->|4. Ready| FinalState(Finished Run)
    
    User -->|5. ds apply| FinalState
    FinalState -- Merge Back --> HumanNode
```

1.  **You Code**: Work in your Human Node.
2.  **Agents React**: On every commit, DevSwarm spins up Agent Nodes.
3.  **Parallel Execution**: While you continue coding, Agent 1 writes tests, Agent 2 reviews code.
4.  **Loop Closed**: You use `ds apply` to merge the Agents' work back into your branch when you are ready.

---

## 🚀 Quick Start

### Installation

**One-Click Install (Recommended)**

```bash
curl -fsSL https://raw.githubusercontent.com/bytedance/DevSwarm/main/install.sh | bash
```

**Manual Install**

See [Installation Guide](user-guide/installation.md) for building from source.

### Usage

#### 1. Initialize
```bash
mkdir myproject_swarm && cd myproject_swarm
ds init https://github.com/user/repo.git
```

#### 2. Start Coding (Human Node)
```bash
# Create a node for your feature
ds spawn feature/login login-dev

# Enter the isolated environment
ds enter login-dev
```

#### 3. Agent Collaboration
When you commit code in `login-dev`, a workflow starts automatically.

```bash
# Check agent status
ds workflow ls

# Inspect what the agent did
ds workflow inspect <run-id>

# Merge agent's changes back to your node
ds apply login-dev
```

---

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
