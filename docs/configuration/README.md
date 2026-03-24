# Configuration

Orion has two main configuration surfaces.

## 1) AI config for `orion ai`

File: `~/.orion.conf`

```yaml
api_key: "$MOONSHOT_API_KEY"
base_url: "https://api.moonshot.cn/v1"
model: "kimi-k2-turbo-preview"
```

`api_key` can be a raw key or an environment variable reference.

## 2) Workspace config

File: `.orion/config.yaml`

```yaml
version: 1

git:
  main_branch: main

runtime:
  artifact_dir: .orion/runs

agents:
  default_provider: traecli
  providers:
    traecli:
      command: 'traecli "{{.Prompt}}" -py'
    qwen:
      command: 'qwen "{{.Prompt}}" -y'
    kimi:
      command: 'kimi -y -p "{{.Prompt}}"'

notifications:
  enabled: true
  poll_interval: 5s
  silence_threshold: 20s
  reminder_interval: 5m
  llm_classifier:
    enabled: true
```

## Related workflow files

- `.orion/workflows/*.yaml`
- `.orion/agents/*.yaml`
- `.orion/prompts/*.md`

These define pipeline steps, runtime binding, and prompt behavior.
