# Agentic Workflow 指南

[English](workflow.md) | [简体中文](workflow_zh-CN.md)

Orion 开启了 **Agentic DevOps** 模式，AI Agents 与你并肩工作。Agents 不在远程 CI/CD 流水线中等待，而是在本地独立的节点中运行，通过 **Shadow Branch (影子分支)** 链接它们的工作。

## 1. 工作原理

1.  **触发**: 你通过 CLI 手动触发（或通过特定事件触发）工作流。
2.  **工作流启动**: 自动执行定义的工作流（如 `release-workflow.yaml`）。
3.  **影子分支链**:
    -   **步骤 1 (Agent)**: 创建影子分支，执行任务，例如变基 (Rebase)、解决冲突并提交。
4.  **应用 (Apply/Merge)**: 你或工作流本身将最终结果合并回你的 Human Node。

## 2. 配置

工作流定义在 `.orion/workflows/*.yaml` 中。例如 `release-workflow.yaml`：

```yaml
name: release-workflow

trigger:
  event: manual # 通过 CLI 手动触发

pipeline:
  - id: rebase
    type: agent
    agent: rebase-agent # 引用 .orion/agents/rebase-agent.yaml
    base-branch: ${input.node.branch}
    
  - id: commit-check
    type: bash
    node: ${steps.rebase.node}
    run: |
      # 检查并确保提交存在
    depends_on: [rebase]
```

## 3. 管理工作流

### 列出运行记录
查看所有活跃和历史工作流运行：

```bash
orion workflow ls
# 输出:
# RUN ID        STATUS   TRIGGER          BASE BRANCH
# run-abc1234   success  commit(a1b2c)    feature/login
```

### 检查运行详情
查看详细步骤和状态：

```bash
orion workflow inspect run-abc1234
```

## 4. 应用更改 (闭环)

一旦 Agent 完成工作（例如修复了 Bug），你需要将这些更改带回你的工作分支。

**不要手动使用 `git merge`。** 请使用 `orion apply`：

```bash
# 1. 检查节点状态
orion inspect login-node

# 2. 应用工作流结果
orion apply login-node
```

系统会提示你选择要应用的工作流运行。Orion 随后会将最终的 Shadow Branch 合并到你的 Human Node 工作区中。

## 5. 手动触发

你也可以在不提交代码的情况下手动触发工作流：

```bash
orion workflow run default
```
