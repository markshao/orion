# <img src="assets/icon.svg" alt="Orion Logo" width="40" height="40" align="top"/> Orion

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

Orion is an AI-native development environment manager for parallel coding agents.
It combines Git worktrees, tmux sessions, and agent workflows into one local protocol.

## Why Orion

- Parallel and isolated execution: each node has its own worktree, session, and branch context.
- Agent-first Git model: humans manage goals; agents handle branch-heavy routine operations.
- Human-as-manager workflow: enter any node at any time to inspect progress and redirect work.
- Local Agentic DevOps: run integration steps locally with agent pipelines, not only in remote CI.
- Attention routing: Orion can detect pending input in node sessions and notify you.
- Official Feishu/Lark support: route waiting-input events to team chat for faster human feedback loops.

## Quick Start (5 Minutes)

This path covers your target loop: create node with AI, code, detach, switch to new work, return on notification, then push safely.

### 1) Install latest and initialize

```bash
curl -fsSL https://raw.githubusercontent.com/markshao/orion/main/install.sh | bash
```

Then initialize a workspace:

```bash
# create a workspace
orion init https://github.com/<you>/<repo>.git
cd <repo>_swarm
```

### 2) Let Orion AI create your first human node

```bash
orion ai "implement login API and session flow"
orion enter <node-name>
```

Inside tmux, launch your coding agent and start vibe coding:

```bash
codex
# or
kimi
```

Detach from tmux:

```text
Ctrl+b, then d
```

### 3) Keep moving: start another feature

```bash
orion ai "start payment retry feature"
```

When Orion detects another node is waiting for your input, you'll receive notifications (including Feishu/Lark, if configured). You can also check watcher status:

```bash
orion notification-service status
orion notification-service list-watchers
```

Jump back to the waiting node:

```bash
orion enter <previous-node-name>
```

### 4) Delegate release workflow and publish

```bash
orion workflow run release-workflow --node <node-name>
orion workflow ls
orion inspect <node-name>
```

Then finish commit/rebase/push with the Codex skill:

```text
/push_remote
```

`/push_remote` requires the [`push-remote` skill](orion-skills/README.md) to be installed in Codex.

Or use Orion native push directly:

```bash
orion push <node-name>
```

## Core Commands

| Command | Purpose |
| --- | --- |
| `orion init <repo-url>` | Initialize an Orion workspace |
| `orion ai "<task>"` | Create branch + node from natural language |
| `orion spawn <branch> <node>` | Manually create a node |
| `orion enter <node>` | Enter the node tmux session |
| `orion ls` | List active nodes |
| `orion inspect <node>` | Show node status and branch details |
| `orion workflow run <workflow> --node <node>` | Trigger an agent workflow |
| `orion workflow ls` | List workflow runs |
| `orion workflow inspect <run-id>` | Inspect workflow details |
| `orion notification-service status` | Show notification service status |
| `orion notification-service list-watchers` | Show watcher state and pending input |
| `orion push <node>` | Push node branch to remote |

## Learn More

- [Concepts: Human Node, Agentic Node, Shadow Branch](docs/concepts/README.md)
- [Workflows: Local Agentic DevOps and pipeline config](docs/workflows/README.md)
- [Notifications: waiting-input detection and alerting](docs/notifications/README.md)
- [Configuration: LLM, runtime, and workspace settings](docs/configuration/README.md)
- [Architecture: workspace, git, tmux, workflow engine](docs/architecture/README.md)
- [Installation Guide](user-guide/installation.md)
- [Human Node Guide](user-guide/human-node.md)
- [Orion Skills (`push-remote`)](orion-skills/README.md)

## License

Apache License 2.0
