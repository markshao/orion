# Agentic Workflow Guide

[English](workflow.md) | [简体中文](workflow_zh-CN.md)

Orion enables **Agentic DevOps**, where AI agents work alongside you. Instead of waiting for a remote CI/CD pipeline, agents run locally in their own nodes and operate on **Shadow Branches** to prepare code for publishing.

## 1. How it Works

1.  **Trigger**: You trigger a workflow manually via CLI on a human node.
2.  **Workflow Start**: The workflow (for example `release-workflow.yaml`) is executed.
3.  **Agentic Execution**:
    -   **Agent Step**: Orion creates an agentic node on a shadow branch.
    -   The agent performs tasks such as rebasing, conflict resolution, testing, and preparing commits.
4.  **Node Status Update**: If the workflow succeeds, the human node is marked `READY_TO_PUSH`.
5.  **Publish**: You push the validated branch with `orion push`.

## 2. Configuration

Workflows are defined in `.orion/workflows/*.yaml`. For example, `release-workflow.yaml`:

```yaml
name: release-workflow

trigger:
  event: manual # Triggered manually via CLI

pipeline:
  - id: rebase
    type: agent
    agent: rebase-agent # Refers to .orion/agents/rebase-agent.yaml
    base-branch: ${input.node.branch}
    
  - id: commit-check
    type: bash
    node: ${steps.rebase.node}
    run: |
      # Check and ensure commits
    depends_on: [rebase]
```

## 3. Managing Workflows

### List Runs
View all active and past workflow runs:

```bash
orion workflow ls
# Output:
# RUN ID                 WORKFLOW           TRIGGER   STATUS    BASE BRANCH
# run-20260321-abcd1234  release-workflow   manual    success   feature/login
```

### Inspect a Run
See detailed steps and status:

```bash
orion workflow inspect run-abc1234
```

## 4. Release Loop

Once the workflow finishes successfully, Orion updates the human node status so you know the branch is ready to publish.

Recommended flow:

```bash
# 1. Run the release workflow on your human node
orion workflow run release-workflow --node login-node

# 2. Check the node status
orion inspect login-node

# 3. Push when the node becomes READY_TO_PUSH
orion push login-node
```

In the built-in `release-workflow`, Orion uses an agentic node to help with rebase and conflict handling before marking the human node as `READY_TO_PUSH`.

## 5. Manual Trigger

You can trigger a workflow manually on a specific node at any time:

```bash
orion workflow run release-workflow --node login-node
```
