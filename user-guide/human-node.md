# Human Node Guide

[English](human-node.md) | [简体中文](human-node_zh-CN.md)

In Orion, a **Human Node** is your primary workspace. It maps to a Git Worktree and a Tmux Session, providing an isolated environment for your manual coding tasks.

## 1. Initialize Workspace

Initialize Orion for your project:

```bash
orion init https://github.com/user/repo.git
```

This command:
- Clones the repository as a bare Git store in `repo.git/`.
- Creates `.orion/` configuration directory.

Orion keeps editable code only in node worktrees. The bare repo stores refs, objects, tags, and remote state.

## 2. Create a Human Node

Create a new node to work on a specific feature.

**Recommended: use natural language**

```bash
orion ai "implement login flow"
```

This lets Orion generate a branch name and node name for you automatically.

**Manual option**

```bash
# Syntax: orion spawn <branch> <node-name> --base <base-branch>
orion spawn feature/login login-node --base main --label "Frontend"
```

- **Branch**: The logical branch you want to work on (e.g., `feature/login`).
- **Node Name**: A unique name for this environment (e.g., `login-node`).
- **--base**: The base branch to branch off from if `feature/login` doesn't exist.

## 3. Enter the Node

Start coding by entering the node's environment:

```bash
orion enter login-node
```

This will:
- Attach you to a dedicated **Tmux Session**.
- Change the directory to the node's **Git Worktree**.
- Isolate your process from other nodes.

## 4. VSCode Integration

Orion automatically maintains a `Orion.code-workspace` file in the root.

1. Open this workspace file in VSCode: `code Orion.code-workspace`.
2. All your active nodes (Worktrees) will appear as root folders in the Explorer.
3. You can edit files across multiple nodes simultaneously in one window.

## 5. Release Workflow and Push

The recommended human-node flow is:

```bash
# 1. Develop and commit in the node
orion enter login-node
git add .
git commit -m "feat: implement login flow"

# 2. Run the release workflow
orion workflow run release-workflow --node login-node

# 3. Inspect status
orion inspect login-node

# 4. Push when status becomes READY_TO_PUSH
orion push login-node
```

The `release-workflow` creates agentic nodes on shadow branches to help with rebase and conflict resolution before the branch is pushed.

## 6. Bare Repo Operations

Use `orion run` without `-w` for Git-only operations against the bare repo, especially release tasks:

```bash
orion run git fetch origin
orion run git tag v1.0.0
orion run git push origin v1.0.0
orion run git push --tags
```

Use `orion run -w <node>` for commands that need a working tree:

```bash
orion run -w login-node go test ./...
orion run -w login-node make build
```

## 7. Cleaning Up

When you are done with a task:

```bash
orion rm login-node
```

**Safety Check**: If the node still has successful workflow runs that have not yet been pushed, `orion rm` will warn you to help prevent losing work.
