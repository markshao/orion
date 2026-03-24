# Workflows

Workflow 让 Orion 在本地执行 agentic pipeline，处理例行 DevOps 工作。

[English](README.md) | [简体中文](README_zh-CN.md)

## Local Agentic DevOps

<img src="../../assets/diagrams/local-agentic-devops.png" alt="Local Agentic DevOps diagram" width="900" />

典型流程：

1. 人类在 Human Node 开发并提交。
2. 运行 `orion workflow run release-workflow --node <node>`。
3. Orion 在 shadow branch 上创建 agentic nodes。
4. step 按顺序或依赖关系执行。
5. 结果合并回 Human Node。
6. 节点就绪后由人类推送。

## 为什么需要 Workflow

- 把 rebase、冲突处理、检查等重复工作交给 agent
- 每个 step 都隔离且可复现
- 从单 agent 平滑扩展到多 step / 多 agent pipeline

## 如何配置 Workflow

Workflow 文件位于 `.orion/workflows/*.yaml`。

最小示例：

```yaml
name: release-workflow

trigger:
  event: manual

pipeline:
  - id: rebase
    type: agent
    agent: rebase-agent
    base-branch: ${input.node.branch}

  - id: merge
    type: bash
    node: ${input.node}
    run: |
      git merge ${ORION_TARGET_BRANCH} --no-edit
    depends_on: [rebase]
```

相关文件：

- `.orion/agents/*.yaml`：agent 运行时绑定
- `.orion/prompts/*.md`：step prompt 定义

## 常用命令

```bash
orion workflow run <workflow> --node <node>
orion workflow ls
orion workflow inspect <run-id>
orion workflow enter <run-id> <step-id>
orion workflow artifacts ls <run-id>
orion workflow logs <run-id> [step-id]
```
