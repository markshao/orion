# Agentic Workflow Guide

Orion enables **Agentic DevOps** by orchestrating AI agents to work concurrently with you in isolated environments. This guide details how to configure custom workflows, define specialized agents, and leverage the "Chain of Branch" architecture.

---

## 1. Core Concepts

### The "Chain of Branch" Model
Orion treats every agent task as a Git branch operation.

1.  **Human Node**: You work on a feature branch (e.g., `feature/login`).
2.  **Shadow Branch**: When a workflow starts, Orion creates a temporary branch off your current work (e.g., `orion/run-123/ut`).
3.  **Agent Execution**: The agent works in this shadow branch—writing code, running tests, or fixing bugs.
4.  **Chaining**: Multiple agents can be chained. Agent B can start from Agent A's shadow branch (`depends_on`).
5.  **Merge Back**: You review the final result and merge the agent's work back into your feature branch using `orion apply`.

### Artifacts vs. Code
*   **Code**: Changes to source files (`*.go`, `*.py`) are committed to the shadow branch.
*   **Artifacts**: Non-code outputs (reports, logs, binaries) are written to a special `ArtifactDir` and are **not** committed to Git, but are persisted in Orion's run storage.

---

## 2. Configuring Custom Workflows

Workflows are defined in `.orion/workflows/*.yaml`. They describe the *pipeline* of tasks.

**Example: Unit Test -> Code Review Pipeline**
Create `.orion/workflows/ut-cr.yaml`:

```yaml
name: ut-cr-workflow

# Trigger Configuration
trigger:
  event: manual      # Options: 'manual' (CLI), 'commit' (Auto)
  # branch: feature/* # Optional: Only trigger on specific branches

pipeline:
  # Step 1: Unit Test Generation
  - id: ut-gen           # Unique ID for this step
    agent: ut-agent      # Refers to .orion/agents/ut-agent.yaml
    suffix: ut           # Shadow branch suffix: orion/<run-id>/ut

  # Step 2: Code Review
  - id: code-review
    agent: cr-agent      # Refers to .orion/agents/cr-agent.yaml
    depends_on: [ut-gen] # CR starts from the result of 'ut-gen'
    suffix: cr           # Shadow branch suffix: orion/<run-id>/cr
```

### Dependency Logic
*   If `depends_on` is set, the step's shadow branch is created from the dependency's shadow branch.
*   If `depends_on` is empty (default), the step starts from the **Base Branch** (your current code).

---

## 3. Configuring Custom Agents

Agents are the "workers". They are defined in `.orion/agents/*.yaml`.

**Example: A Unit Test Agent**
Create `.orion/agents/ut-agent.yaml`:

```yaml
name: ut-agent

runtime:
  # The AI Provider to use (configured in .orion/config.yaml)
  # Built-ins: 'traecli', 'qwen'
  provider: qwen
  
  # Optional: Specific model parameters
  # model: qwen-max
  # params:
  #   temperature: 0.7

# The instruction file for the agent
prompt: ut-gen.md
```

### Runtime Providers
Orion delegates execution to providers. By default:
*   `traecli`: Uses the `traecli` binary with python mode (`-py`).
*   `qwen`: Uses `traecli` binary with auto-yes mode (`-y`).
*   **Custom**: You can define your own providers in `.orion/config.yaml` with custom command templates (e.g., `docker run ...`).

---

## 4. Writing Effective Prompts

Prompts (`.orion/prompts/*.md`) tell the agent *what* to do. Orion injects context variables at runtime.

**Example: `ut-gen.md`**

```markdown
# Unit Test Generation

## Objective
Generate comprehensive unit tests for the modified code in this branch.

## Context Variables
Orion automatically replaces these:
- `{{.Branch}}`: Current shadow branch name.
- `{{.ArtifactDir}}`: Absolute path to store reports/logs.
- `{{.CommitID}}`: Current commit hash.

## Instructions
1.  **Analyze**: Look at the `git diff` to understand changes.
2.  **Generate**: Write Go test cases in `*_test.go` files.
3.  **Report**: Write a summary of coverage to `{{.ArtifactDir}}/coverage_report.md`.

## Rules
- Do NOT modify the original source code, only add tests.
- Ensure all tests compile and run.
```

---

## 5. Artifact Management

When an agent needs to produce output that isn't code (e.g., a security scan report, a compiled binary, or a log file), it **must** write to `{{.ArtifactDir}}`.

### Why?
*   **Persistence**: Artifacts are stored outside the Git worktree (`.orion/runs/...`), so they survive even if the shadow branch is deleted.
*   **Visibility**: You can easily inspect them via CLI without switching branches.

### Usage
1.  **Agent**: Writes to `{{.ArtifactDir}}/my-report.txt`.
2.  **User**: Inspects via:
    ```bash
    orion workflow artifacts ls <run-id>
    ```
    Output:
    ```
    📂 Step: code-review
      - my-report.txt  (/absolute/path/to/report.txt)
    ```

---

## 6. Advanced: Environment & Auth

Orion automatically injects your authentication context into the agent's environment so it can access private resources:

*   **SSH**: `SSH_AUTH_SOCK` is forwarded (for cloning private Git repos).
*   **Kerberos**: `KRB5CCNAME` is forwarded (for internal service authentication).
*   **Custom Env**: You can define `env` in `agent.yaml`:
    ```yaml
    env:
      - GOPROXY=https://goproxy.io
    ```
