# Concepts

Orion introduces a local execution protocol for humans and coding agents.

## Logical Branch

The branch humans reason about, such as `feature/login`.

## Human Node

A long-lived node owned by a human task. It includes:

- an isolated Git worktree
- a dedicated tmux session
- a branch context for iterative development

Think of it as a managed "developer workstation" you can enter at any time.

## Agentic Node

An ephemeral node created by workflow steps.

- runs an agent in isolation
- usually executes on a shadow branch
- can be chained to another step as input

## Shadow Branch

A system-managed branch used for safe experimentation and step chaining.

Pattern example:

```text
orion/<logical-branch>/<node-or-step>
```

Workflow steps can pass results by using previous step shadow branches as their base.

## Why this model

- isolates concurrent agent work
- keeps risky integration operations away from your main dev node
- lets humans supervise many nodes without handling Git internals manually
