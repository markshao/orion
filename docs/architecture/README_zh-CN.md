# Architecture

Orion 是一个面向并行 coding nodes 的本地控制平面。

[English](README.md) | [简体中文](README_zh-CN.md)

## 分层

1. Workspace Manager（`internal/workspace`）
- 管理 `.orion/state.json` 和工作区元数据
- 管理 node 生命周期：spawn、enter、inspect、remove

2. Git Provider（`internal/git`）
- 封装 Git 操作（`clone`、`worktree`、`branch`、`push`）
- 管理 node/workflow 的 logical/shadow branch 流程

3. Tmux Provider（`internal/tmux`）
- 创建与接管 node session
- 提供进程级隔离

4. Workflow Engine（`internal/workflow`）
- 执行 pipeline step（`agent` / `bash`）
- 在依赖 step 之间链式传递 shadow branch
- 写入 artifacts 和 run 元数据

5. Notification Service（`internal/notification`）
- 观察 node pane 输出
- 分类等待输入信号
- 向用户分发通知

## 执行模型

- Human Node：长期、面向任务。
- Workflow Node：短期、面向步骤。
- 分支交接：通过 shadow branch 完成。
- 最终结果：合并回目标节点后推送。
