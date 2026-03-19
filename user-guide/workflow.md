# Agentic Workflow Guide

[English](workflow.md) | [简体中文](workflow_zh-CN.md)

Orion enables **Agentic DevOps**, where AI agents work alongside you. Instead of waiting for a remote CI/CD pipeline, agents run locally in their own nodes, chaining their work on **Shadow Branches**.

## 1. How it Works

1.  **Trigger**: You trigger a workflow manually via CLI (or upon certain events).
2.  **Workflow Start**: The workflow (e.g., `release-workflow.yaml`) is executed.
3.  **Shadow Branch Chain**:
    -   **Step 1 (Agent)**: Creates a shadow branch, performs tasks like rebasing, resolving conflicts, and creating commits.
4.  **Apply/Merge**: You or the workflow itself merges the final result back to your Human Node.

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
# RUN ID        STATUS   TRIGGER          BASE BRANCH
# run-abc1234   success  commit(a1b2c)    feature/login
```

### Inspect a Run
See detailed steps and status:

```bash
orion workflow inspect run-abc1234
```

## 4. Applying Changes (The Loop)

Once an agent has finished its work (e.g., fixed a bug), you need to bring those changes back to your working branch.

**Do NOT use `git merge` manually.** Use `orion apply`:

```bash
# 1. Check your node status
orion inspect login-node

# 2. Apply the workflow result
orion apply login-node
```

You will be prompted to select which workflow run to apply. Orion will then merge the final Shadow Branch into your Human Node's worktree.

## 5. Manual Trigger

You can also trigger a workflow manually without committing:

```bash
orion workflow run default
```
