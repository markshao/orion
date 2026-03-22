# Agentic Workflow 指南

[English](workflow.md) | [简体中文](workflow_zh-CN.md)

Orion 开启了 **Agentic DevOps** 模式，AI Agents 与你并肩工作。Agents 不在远程 CI/CD 流水线中等待，而是在本地独立节点中运行，并通过 **Shadow Branch (影子分支)** 为发布前的集成工作做准备。

## 1. 工作原理

1.  **触发**: 你在 human node 上通过 CLI 手动触发 workflow。
2.  **工作流启动**: 执行定义好的工作流，例如 `release-workflow.yaml`。
3.  **Agentic 执行**:
    -   **Agent 步骤**: Orion 创建基于 shadow branch 的 agentic node。
    -   Agent 执行 rebase、冲突处理、测试以及提交准备等任务。
4.  **节点状态更新**: 如果 workflow 成功，human node 会被标记为 `READY_TO_PUSH`。
5.  **发布**: 你通过 `orion push` 推送验证后的分支。

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
# RUN ID                 WORKFLOW           TRIGGER   STATUS    BASE BRANCH
# run-20260321-abcd1234  release-workflow   manual    success   feature/login
```

### 检查运行详情
查看详细步骤和状态：

```bash
orion workflow inspect run-abc1234
```

## 4. 发布闭环

当 workflow 成功结束后，Orion 会更新 human node 状态，告诉你这个分支已经准备好发布。

推荐流程：

```bash
# 1. 在你的 human node 上运行 release workflow
orion workflow run release-workflow --node login-node

# 2. 检查节点状态
orion inspect login-node

# 3. 当节点变成 READY_TO_PUSH 后执行 push
orion push login-node
```

在内置的 `release-workflow` 中，Orion 会使用 agentic node 帮你完成 rebase 和冲突处理，然后把 human node 标记为 `READY_TO_PUSH`。

## 5. 手动触发

你可以在任意时机对指定 node 手动触发 workflow：

```bash
orion workflow run release-workflow --node login-node
```
