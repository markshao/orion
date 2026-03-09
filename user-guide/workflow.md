# Agentic Workflow Guide

[English](workflow.md) | [简体中文](workflow_zh-CN.md)

Orion enables **Agentic DevOps**, where AI agents work alongside you. Instead of waiting for a remote CI/CD pipeline, agents run locally in their own nodes, chaining their work on **Shadow Branches**.

## 1. How it Works

1.  **Trigger**: You commit code in your Human Node.
2.  **Workflow Start**: A workflow (defined in `default.yaml`) is triggered automatically.
3.  **Shadow Branch Chain**:
    -   **Step 1 (UT Agent)**: Creates a shadow branch off your commit, runs unit tests, fixes bugs, and commits changes.
    -   **Step 2 (Review Agent)**: Creates a new shadow branch off Step 1's result, reviews code, adds comments or refactors.
4.  **Apply**: You review the final result and merge it back to your Human Node.

## 2. Configuration

Workflows are defined in `.orion/workflows/default.yaml`.

```yaml
name: default
trigger:
  event: commit # Trigger on every commit

pipeline:
  - id: ut
    agent: ut-agent # Refers to .orion/agents/ut-agent.yaml
    suffix: ut

  - id: cr
    agent: cr-agent
    depends_on: [ut] # Runs after UT agent finishes
    suffix: cr
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
