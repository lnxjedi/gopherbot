# Slack Socket Mode

Slack is configured in `custom/conf/protocols/slack.yaml`.

Minimal shape:

```yaml
ProtocolConfig:
  AppToken: xapp-{{ decrypt "<slack-app-token>" }}
  BotToken: xoxb-{{ decrypt "<slack-bot-token>" }}
  HearSelf: false
  UserMap:
    alice: U12345678
```

## Notes

- `UserMap` is connector-local in v3. It does not belong in `robot.yaml`.
- The engine still makes authorization decisions by canonical username, not by Slack user ID.
- Slash commands arrive as hidden bot-addressed messages, so hidden commands still have to be explicitly allowed by the plugin.

If Slack is your primary protocol, set it in `custom/conf/environments/<environment>.yaml` or directly in `custom/conf/robot.yaml`, depending on how you structure your environments.
