# 概念

Orion 为人类与 Coding Agent 提供一套本地执行协议。

[English](README.md) | [简体中文](README_zh-CN.md)

## Logical Branch（逻辑分支）

人类认知中的业务分支，例如 `feature/login`。

## Human Node（人类节点）

面向人类任务的长期节点，包含：

- 独立的 Git worktree
- 独立的 tmux session
- 可持续迭代的分支上下文

它可以理解成“可随时进入的开发者工作站”。

## Agentic Node（智能体节点）

由 workflow step 创建的临时节点：

- 在隔离环境里运行 agent
- 通常在 shadow branch 上执行
- 可作为后续 step 的输入

## Shadow Branch（影子分支）

系统管理的分支，用于安全实验和 step 间结果传递。

命名示例：

```text
orion/<logical-branch>/<node-or-step>
```

workflow 可以基于前一 step 的 shadow branch 继续执行，实现链式传递。

## 这套模型的价值

- 支持多 agent 并发且互不干扰
- 让高风险集成操作远离主开发节点
- 人可以管理多个节点而不必手工处理大量 Git 细节
