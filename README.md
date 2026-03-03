# DevSwarm 🐝

**DevSwarm** is an AI-native development environment manager designed to facilitate concurrent collaboration between humans and AI agents.

In the era of AI Coding, traditional Git workflows struggle to support multi-task concurrency on the same logical branch. DevSwarm solves this by virtualizing Git branches into **Nodes**, where each node provides an isolated environment (Worktree + Tmux Session) for humans or agents to work independently without conflict.

## 🚀 Key Features

- **Node Abstraction**: Shields the complexity of `git worktree`, providing a clean interface to manage concurrent development units.
- **Environment Isolation**: Each Node has its own file system (Worktree) and runtime session (Tmux), ensuring zero interference.
- **Shadow Branching**: Automatically manages `shadow branches` (e.g., `devswarm/login-test/feature/login`) to allow parallel work on the same logical feature.
- **AI-Ready**: Designed as infrastructure for AI Agents, allowing them to spawn, code, and test in their own sandboxed nodes.
- **State Management**: Persists workspace state to survive restarts and crashes.

## 📦 Installation

We provide an automated installation script that downloads the latest binary release from GitHub and configures autocompletion.

```bash
curl -sL https://raw.githubusercontent.com/markshao/DevSwarm/main/install.sh | bash
```

This will:

1. Detect your OS and Architecture.
2. Download the latest binary from [GitHub Releases](https://github.com/markshao/DevSwarm/releases).
3. Install `devswarm` to `/usr/local/bin`.
4. Configure autocompletion for your shell (Zsh/Bash).

## 🎮 Playground (Try it out!)

DevSwarm comes with an automated **End-to-End (E2E) Test Script** that simulates a full user workflow. This is the best way to understand how DevSwarm works without polluting your environment.

```bash
# Run the E2E test script
./playground/test_e2e.sh
```

This script will:

- Initialize a workspace.
- Spawn a node.
- Simulate coding work.
- Merge changes back.
- Clean up resources.

## 🛠 Usage Guide

### 1. Initialize Workspace

Create a DevSwarm workspace for a Git repository.

```bash
# Automatically creates 'repo_workspace' directory (e.g. DevSwarm_workspace)
devswarm init https://github.com/markshao/DevSwarm.git

# Or specify a custom directory name
devswarm init https://github.com/markshao/DevSwarm.git my_custom_workspace
```

> **Note**: All subsequent commands must be run inside the workspace directory.

### 2. Spawn Nodes

Create isolated nodes for concurrent tasks.

```bash
cd DevSwarm_workspace

# Spawn a node for a new feature (creates branch from main if missing)
devswarm spawn feature/login login-dev --base main --purpose coding

# Spawn another node for testing the same feature concurrently
devswarm spawn feature/login login-test --base feature/login --purpose testing
```

### 3. Enter Node (Tmux)

Jump into the isolated development environment (Tmux Session) of a node.

```bash
devswarm enter login-dev
```

_You are now inside a Tmux session. The working directory is `nodes/login-dev`. Use `Ctrl+b, d` to detach._

### 4. List Nodes

Check the status of all active nodes.

```bash
devswarm ls
```

_Output:_

```text
NODE          BRANCH         PURPOSE      SESSION   CREATED
login-dev     feature/login  coding       RUNNING   2023-10-27T10:00:00Z
login-test    feature/login  testing      STOPPED   2023-10-27T10:05:00Z
```

### 5. Merge Node

Merge the changes from a node (Shadow Branch) back to the main Logical Branch.

```bash
# Squash merge and keep the node
devswarm merge login-dev

# Squash merge and automatically remove the node
devswarm merge login-dev --cleanup
```

### 6. Remove Node

Manually remove a node and release all resources (Worktree, Branch, Session).

```bash
devswarm rm login-test
```

## 💻 IDE Integration (VS Code / Trae)

DevSwarm can automatically attach your IDE terminal to the correct Node's Tmux session based on the file you are currently editing.

### Setup Instructions

1.  Open your VS Code `settings.json`.
2.  Add or replace the following configuration:

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

> **Note**: We use the `CURRENT_FILE` environment variable because VS Code argument expansion (`${file}`) can sometimes behave inconsistently in certain contexts.

### How it works

- When you open a terminal (`Ctrl + ~`), DevSwarm detects if the active file (passed via `CURRENT_FILE`) belongs to a specific **Node**.
- If it does, you are automatically attached to that node's **Tmux session**.
- If not, you are attached to a **default** session.

## 🏗 Architecture

- **Repo**: The single source of truth (Git repository).
- **Nodes**: Ephemeral worktrees derived from the repo.
- **State**: A `state.json` file tracks the mapping between Nodes, Tmux sessions, and Git branches.

```text
my_project_swarm/
├── .devswarm/
│   └── state.json
├── repo/                # Main .git repository
└── nodes/
    ├── login-human/     # Worktree A
    └── login-test/      # Worktree B
```

## 📝 License

MIT
