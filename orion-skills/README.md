# Orion Skills

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

`orion-skills/` stores reusable agent skills that are maintained as part of the Orion product.

The goal is simple:

- Keep Orion-specific skills versioned inside the Orion repository
- Make each skill easy to inspect, copy, and install into a user's coding agent
- Leave room for more skills over time without mixing them into core CLI code

## Directory Layout

Each skill lives in its own folder:

```text
orion-skills/
├── README.md
└── push-remote/
    ├── SKILL.md
    └── agents/
        └── openai.yaml
```

Recommended conventions:

- One directory per skill
- `SKILL.md` as the primary behavior contract
- `agents/` for agent-specific UI or launcher metadata
- Keep skill content self-contained and copyable

## Included Skills

### `push-remote`

`push-remote` is a Codex-oriented skill for finishing the last mile of a feature branch safely:

- inspect git state
- format with repo-native tooling
- create a clean commit
- rebase onto the latest `origin/main`
- run the existing tests
- push the branch conservatively

## Install Into Codex

Codex loads local skills from `~/.codex/skills/`.

To install `push-remote` from this repository:

```bash
mkdir -p ~/.codex/skills
cp -R ./orion-skills/push-remote ~/.codex/skills/push-remote
```

If you are already inside the Orion repository root, that is enough. After that, start a new Codex session or reload the environment if your setup caches skill discovery.

## Use In Codex

Once installed, invoke it with either of these forms:

```text
/push_remote
```

```text
Use $push-remote to commit the current work and push it.
```

The canonical skill name is `push-remote`.

## Update A Skill

When the repository version changes, reinstall by replacing the local copy:

```bash
rm -rf ~/.codex/skills/push-remote
cp -R ./orion-skills/push-remote ~/.codex/skills/push-remote
```

## Notes

- This directory is repository-owned source of truth for Orion-distributed skills.
- The current example targets Codex because its local skill layout is simple and explicit.
- If Orion later supports first-class skill installation, this directory can remain the source package layout.
