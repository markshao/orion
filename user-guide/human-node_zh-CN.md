# Human Node 指南

[English](human-node.md) | [简体中文](human-node_zh-CN.md)

在 Orion 中，**Human Node (人类节点)** 是你的主要工作区。它对应一个 Git Worktree 和一个 Tmux Session，为你的人工编码任务提供隔离的环境。

## 1. 初始化工作区

为你的项目初始化 Orion：

```bash
orion init https://github.com/user/repo.git
```

此命令会：
- 将主仓库克隆到 `main_repo/`。
- 创建 `.orion/` 配置目录。

## 2. 创建 Human Node

创建一个新节点来开发特定功能。

**推荐：使用自然语言创建**

```bash
orion ai "实现登录流程"
```

这样 Orion 会自动帮你生成 branch 名和 node 名。

**手动创建**

```bash
# 语法: orion spawn <branch> <node-name> --base <base-branch>
orion spawn feature/login login-node --base main --label "Frontend"
```

- **Branch**: 你想要工作的逻辑分支 (例如 `feature/login`)。
- **Node Name**: 该环境的唯一名称 (例如 `login-node`)。
- **--base**: 如果 `feature/login` 不存在，则基于此分支创建。

## 3. 进入节点 (Enter)

进入节点环境开始编码：

```bash
orion enter login-node
```

这将：
- 将你 Attach 到专属的 **Tmux Session**。
- 将目录切换到节点的 **Git Worktree**。
- 将你的进程与其他节点隔离。

## 4. VSCode 集成

Orion 会自动在根目录维护一个 `Orion.code-workspace` 文件。

1. 在 VSCode 中打开此文件：`code Orion.code-workspace`。
2. 所有活跃的 Human Node (Worktree) 将作为根文件夹出现在资源管理器中。
3. 你可以在一个窗口中同时编辑多个节点的文件。

## 5. Release Workflow 与 Push

推荐的人类节点工作流如下：

```bash
# 1. 在节点中开发并提交
orion enter login-node
git add .
git commit -m "feat: implement login flow"

# 2. 运行 release workflow
orion workflow run release-workflow --node login-node

# 3. 检查状态
orion inspect login-node

# 4. 当状态变成 READY_TO_PUSH 后执行 push
orion push login-node
```

`release-workflow` 会在 shadow branch 上创建 agentic nodes，帮助你完成 rebase 和冲突处理，然后再进入 push 阶段。

## 6. 清理

当你完成任务后：

```bash
orion rm login-node
```

**安全检查**: 如果该节点还有成功执行但尚未推送的 workflow 结果，`orion rm` 会发出警告，以帮助你避免丢失工作。
