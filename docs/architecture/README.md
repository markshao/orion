# Architecture

Orion is built as a local control plane for parallel coding nodes.

## Layers

1. Workspace Manager (`internal/workspace`)
- owns `.orion/state.json` and workspace metadata
- manages node lifecycle: spawn, enter, inspect, remove

2. Git Provider (`internal/git`)
- wraps Git operations (`clone`, `worktree`, `branch`, `push`)
- manages logical/shadow branch operations for nodes and workflows

3. Tmux Provider (`internal/tmux`)
- creates and attaches node sessions
- provides process isolation per node

4. Workflow Engine (`internal/workflow`)
- runs pipeline steps (`agent` / `bash`)
- chains shadow branches between dependent steps
- writes artifacts and run metadata

5. Notification Service (`internal/notification`)
- observes node panes
- classifies waiting-input signals
- sends user-facing notifications

## Execution model

- Human node is long-lived and task-oriented.
- Workflow node is ephemeral and step-oriented.
- Branch handoff happens through shadow branches.
- Final output is merged back and then pushed.
