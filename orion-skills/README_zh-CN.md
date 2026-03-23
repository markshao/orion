# Orion Skills

[**English**](README.md) | [**简体中文**](README_zh-CN.md)

`orion-skills/` 用来存放作为 Orion 产品体系一部分统一维护的可复用 agent skills。

目标很直接：

- 把 Orion 相关的 skills 直接纳入 Orion 仓库版本管理
- 让每个 skill 都便于查看、复制和安装到用户自己的 coding agent
- 为后续扩展更多 skills 预留统一目录，而不是把这类资产混进核心 CLI 代码

## 目录结构

每个 skill 使用独立目录：

```text
orion-skills/
├── README.md
├── README_zh-CN.md
└── push-remote/
    ├── SKILL.md
    └── agents/
        └── openai.yaml
```

建议遵循这些约定：

- 一个 skill 对应一个目录
- `SKILL.md` 作为主要行为定义文件
- `agents/` 用来放 agent 侧的界面或启动元数据
- skill 内容尽量自包含，便于直接复制安装

## 当前内置 Skill

### `push-remote`

`push-remote` 是一个面向 Codex 的 skill，用于安全完成特性分支的最后一段流程：

- 检查 git 状态
- 使用仓库原生格式化工具整理代码
- 生成清晰的 commit
- rebase 到最新的 `origin/main`
- 运行仓库已有测试
- 保守地推送分支

## 安装到 Codex

Codex 默认从 `~/.codex/skills/` 加载本地 skills。

从当前仓库安装 `push-remote`：

```bash
mkdir -p ~/.codex/skills
cp -R ./orion-skills/push-remote ~/.codex/skills/push-remote
```

如果你当前就在 Orion 仓库根目录下，执行上面两行就够了。安装后，建议开启一个新的 Codex 会话，或者重新加载你的环境，避免 skill 缓存导致识别不到最新内容。

## 在 Codex 中使用

安装完成后，可以用下面任意一种方式触发：

```text
/push_remote
```

```text
Use $push-remote to commit the current work and push it.
```

这个 skill 的规范名称是 `push-remote`。

## 更新 Skill

当仓库里的 skill 有更新时，可以直接覆盖本地版本：

```bash
rm -rf ~/.codex/skills/push-remote
cp -R ./orion-skills/push-remote ~/.codex/skills/push-remote
```

## 说明

- 这个目录是 Orion 对外分发 skills 的仓库内源码来源。
- 当前示例以 Codex 为主，因为它的本地 skill 目录结构简单且明确。
- 如果未来 Orion 自己支持一等公民的 skill 安装能力，这个目录仍然可以继续作为源包布局。
