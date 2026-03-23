# Slack Connector Notes

This file captures Slack connector behavior relevant to routing, hidden commands, and help rendering.

## Source Anchors

- Registration/init: `connectors/slack/static.go`, `connectors/slack/connect.go`
- Incoming message normalization: `connectors/slack/incomingMsgs.go`
- Outgoing + help formatting hooks: `connectors/slack/connectorMethods.go`

## Identity Mapping

- Slack connector identity mapping is connector-local in `ProtocolConfig.UserMap` (`username -> Slack user ID`).
- Connector config mapping is treated as canonical when username/ID collisions exist.
- Engine policy checks remain username-based against global `UserRoster`.

## Inbound Message Normalization

- Standard channel/DM messages are passed as canonical `ConnectorMessage` values with:
  - `BotMessage=false`
  - `HiddenMessage=false`
  - `DirectMessage` set from Slack channel type (`IM` vs channel).
- Slash command events routed to this app are passed as:
  - `BotMessage=true`
  - `HiddenMessage=true`
  - `MessageText=<slash payload text>`
  - no thread metadata (slash commands are non-threaded inbound).

## Hidden Command Semantics

- Registration marks Slack hidden-command support in `robot.ConnectorCapabilities{HiddenCommands: true}`.
- Slack slash commands are platform-routed to one bot app, so connector sets `BotMessage=true`.
- Engine hidden-command policy then treats slash payload as addressed-to-robot without requiring an explicit robot name in text.
- Command still must be explicitly allowed by plugin `AllowedHiddenCommands`.
- Help rendering does not invent a `/(bot)` example for Slack because the actual slash command name is connector/app configuration, not the bot name.
- Instead, Slack exposes a connector-owned generic hint describing hidden-command use through the configured slash command.

## Outgoing Format Behavior

- `Raw` keeps legacy Slack-native behavior (including connector-local `@username` handling outside fenced blocks).
- `Variable` disables Slack markdown parsing and escapes reserved text characters.
- `Fixed` wraps output in fenced code blocks.
- `BasicMarkdown` is rendered with connector-local translation rules:
  - Markdown links `[label](https://...)` are converted to Slack link tokens.
  - `@username` mention tokens are resolved against connector user maps when unambiguous.
  - Mention parsing is skipped inside inline code and fenced code blocks.
  - Emoji shortcodes (for example `:white_check_mark:`) are passed through as shortcodes.
  - Unicode emoji are passed through unchanged.

## Help Rendering Hooks

- Slack connector implements:
  - `FormatHelp(string) string` for line-level Slack-friendly formatting
  - `HiddenCommandHint() string` via `robot.HiddenCommandFormatter`
- `DefaultHelp()` now returns no override so the engine can keep protocol-agnostic quick-help text instead of hardcoding Slack slash syntax.
- Built-in help plugin (`builtin-help`) uses these hooks so `help`, `commands`, and `help-all` output remains readable in Slack formatting while only surfacing hidden-command guidance when it is actually valid.

## slack-go v0.17.x Notes

- `api.GetBotInfo` requires `slack.GetBotInfoParameters` (bot + team ID).
- `slackevents.MessageEvent` no longer exposes attachments/timestamps directly; use embedded `Message` payload (`*slack.Msg`) for attachments and timestamps.

## Runtime Lifecycle Notes

- Slack connector runtime state is connector-instance scoped (not package-global).
- Outbound queueing and edited-message dedupe tracking are maintained per connector instance.
- This supports in-process lifecycle operations used by multi-protocol runtime management:
  - `protocol-stop slack`
  - `protocol-start slack`
  - `protocol-restart slack`
  - remove/add `slack` in `SecondaryProtocols` across reloads
