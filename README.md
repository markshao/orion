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

```bash
git clone https://github.com/your-username/devswarm.git
cd devswarm
make build
# Binary is located at ./bin/devswarm
```

## 🛠 Usage

### 1. Initialize Workspace
Create a new DevSwarm workspace for a repository.

```bash
# Creates a directory 'my_project_swarm' and clones the repo
./bin/devswarm init https://github.com/user/repo.git my_project_swarm
cd my_project_swarm
```

### 2. Spawn Nodes
Create isolated nodes for different tasks.

```bash
# Spawn a node for human development
./bin/devswarm spawn feature/login login-human --purpose coding

# Spawn a node for an AI agent to run tests (auto-creates branch from main if missing)
./bin/devswarm spawn feature/login login-test --base main --purpose agent-test
```

### 3. List Nodes
Check the status of all active nodes.

```bash
./bin/devswarm ls
```
*Output:*
```text
NODE          BRANCH         PURPOSE      SESSION   CREATED
login-human   feature/login  coding       STOPPED   2023-10-27T10:00:00Z
login-test    feature/login  agent-test   RUNNING   2023-10-27T10:05:00Z
```

### 4. Enter Node
Jump into the Tmux session of a node to start working.

```bash
./bin/devswarm enter login-human
```

### 5. Cleanup
Remove a node and release all resources (Worktree, Branch, Session).

```bash
./bin/devswarm rm login-test
```

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
