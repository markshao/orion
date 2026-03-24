# Notifications

Orion notification service scans node sessions and detects when an agent is waiting for human input.

## How it works

- Orion tracks tmux pane output for registered node watchers.
- When output is classified as waiting for input, Orion emits a local notification.
- Entering the node acknowledges the pending wait event.
- Notification delivery supports Feishu/Lark bot integration as an official channel.

## Commands

```bash
orion notification-service start
orion notification-service status
orion notification-service list-watchers
orion notification-service stop
```

## Configuration

Notification settings are in `.orion/config.yaml`:

```yaml
notifications:
  enabled: true
  poll_interval: 5s
  silence_threshold: 20s
  reminder_interval: 5m
  similarity_threshold: 0.99
  tail_lines: 80
  llm_classifier:
    enabled: true
```

## Official Channels

- Local desktop notification
- Feishu/Lark bot notification

## Extensible Provider Architecture

This notification architecture is provider-oriented and reusable across chat systems:

- `Watcher` and wait-input classification stay inside Orion core.
- Delivery adapters can target different notification providers.
- Teams can keep the same detection semantics while swapping transport channels.

Contributions are welcome to add official providers for:

- Slack
- Discord

## Contributing: Notification Providers

If you want to add a new provider, keep this boundary:

- Reuse Orion core watcher and wait-input classification logic.
- Add or extend only the delivery adapter for your target platform.
- Keep provider-specific config isolated so existing channels are unaffected.

Good first targets:

- Slack
- Discord
