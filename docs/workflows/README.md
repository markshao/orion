# Workflows

Workflows let Orion run local agentic pipelines for routine DevOps tasks.

## Local Agentic DevOps

<img src="../../assets/diagrams/local-agentic-devops.png" alt="Local Agentic DevOps diagram" width="900" />

Typical path:

1. Human develops and commits in a Human Node.
2. Human runs `orion workflow run release-workflow --node <node>`.
3. Orion creates agentic nodes on shadow branches.
4. Steps run sequentially or by dependency chain.
5. Result is merged back to the Human Node.
6. Human pushes once node is ready.

## Why workflows

- offload repeatable tasks such as rebase, conflict handling, checks
- keep each step isolated and reproducible
- scale from one agent to multi-step, multi-agent pipelines

## Configure a workflow

Workflow files live in `.orion/workflows/*.yaml`.

Minimal example:

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

Related files:

- `.orion/agents/*.yaml` for agent runtime
- `.orion/prompts/*.md` for step prompts

## Commands

```bash
orion workflow run <workflow> --node <node>
orion workflow ls
orion workflow inspect <run-id>
orion workflow enter <run-id> <step-id>
orion workflow artifacts ls <run-id>
orion workflow logs <run-id> [step-id]
```
