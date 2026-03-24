# <img src="assets/icon.svg" alt="Orion Logo" width="40" height="40" align="top"/> Orion

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

Orion 是一个面向并行 Coding Agents 的 AI 原生开发环境管理器。
它把 Git worktree、tmux session 和 agent workflow 统一成一套本地协作协议。

## 为什么是 Orion

- 并发且隔离：每个 node 都有独立 worktree、session 和分支上下文。
- Agent-first Git：人负责目标与决策，Agent 负责高频 Git 流程和例行操作。
- 人是管理者：你可以随时进入任意 node，查看进度并下发新指令。
- Local Agentic DevOps：把集成流程放回本地，由 Agent pipeline 执行。
- 注意力路由：Orion 可检测 node 中“等待输入”状态并通知你。
- 官方 Feishu/Lark 支持：等待输入事件可直达团队会话，加速反馈闭环。

## 快速开始（5 分钟）

这个路径覆盖你要的主循环：AI 建 node、开发、detach、并行新任务、收到通知回到旧 node、最后安全推送。

### 1) 安装最新版本并初始化

```bash
curl -fsSL https://raw.githubusercontent.com/markshao/orion/main/install.sh | bash
```

然后初始化工作区：

```bash
# 创建工作区
orion init https://github.com/<you>/<repo>.git
cd <repo>_swarm
```

### 2) 用 Orion AI 创建第一个 Human Node

```bash
orion ai "实现登录 API 和会话流程"
orion enter <node-name>
```

进入 tmux 后，先启动 code agent，开始 vibe coding：

```bash
codex
# 或
kimi
```

从 tmux detach：

```text
Ctrl+b, 然后 d
```

### 3) 继续并行：再开一个新功能

```bash
orion ai "启动支付重试功能"
```

当 Orion 检测到其他 node 等待你输入时，你会收到通知（配置后可走 Feishu/Lark）。也可以手动查看 watcher 状态：

```bash
orion notification-service status
orion notification-service list-watchers
```

回到需要处理的节点：

```bash
orion enter <previous-node-name>
```

### 4) 触发 workflow 并发布

```bash
orion workflow run release-workflow --node <node-name>
orion workflow ls
orion inspect <node-name>
```

然后用 Codex skill 完成 commit/rebase/push 收口：

```text
/push_remote
```

`/push_remote` 需要先安装 [`push-remote` skill](orion-skills/README_zh-CN.md)。

也可以直接使用 Orion 原生命令：

```bash
orion push <node-name>
```

## 核心命令

| 命令 | 作用 |
| --- | --- |
| `orion init <repo-url>` | 初始化 Orion 工作区 |
| `orion ai "<task>"` | 用自然语言创建 branch + node |
| `orion spawn <branch> <node>` | 手动创建 node |
| `orion enter <node>` | 进入该 node 的 tmux 会话 |
| `orion ls` | 列出活跃 nodes |
| `orion inspect <node>` | 查看 node 状态和分支信息 |
| `orion workflow run <workflow> --node <node>` | 触发 agent workflow |
| `orion workflow ls` | 列出 workflow 运行记录 |
| `orion workflow inspect <run-id>` | 查看 workflow 详情 |
| `orion notification-service status` | 查看通知服务状态 |
| `orion notification-service list-watchers` | 查看 watcher 和 pending input |
| `orion push <node>` | 推送 node 分支到远端 |

## 深入文档

- [概念：Human Node / Agentic Node / Shadow Branch](docs/concepts/README_zh-CN.md)
- [Workflows：Local Agentic DevOps 与 pipeline 配置](docs/workflows/README_zh-CN.md)
- [Notifications：等待输入检测与通知机制](docs/notifications/README_zh-CN.md)
- [Configuration：LLM、运行时与工作区配置](docs/configuration/README_zh-CN.md)
- [Architecture：workspace、git、tmux、workflow 引擎](docs/architecture/README_zh-CN.md)
- [安装指南](user-guide/installation_zh-CN.md)
- [Human Node 指南](user-guide/human-node_zh-CN.md)
- [Orion Skills（`push-remote`）](orion-skills/README_zh-CN.md)

## License

Apache License 2.0
