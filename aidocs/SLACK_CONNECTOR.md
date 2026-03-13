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

- Slack slash commands are platform-routed to one bot app, so connector sets `BotMessage=true`.
- Engine hidden-command policy then treats slash payload as addressed-to-robot without requiring an explicit robot name in text.
- Command still must be explicitly allowed by plugin `AllowedHiddenCommands`.

## Outgoing Format Behavior

- `Raw` keeps legacy Slack-native behavior (including connector-local `@username` handling outside fenced blocks).
- `Variable` sends a Block Kit `section` block with `plain_text` so the visible body preserves the exact message text without Slack markdown/mention/link interpretation.
- `Fixed` sends a Block Kit `rich_text` block using `rich_text_preformatted` so fixed-width output renders as native Slack preformatted content instead of literal triple-backtick fences.
- Block-backed `Variable` / `Fixed` sends still include top-level fallback text for notifications/accessibility and legacy RTM fallback behavior.
- Targeted user-in-channel sends use a readable literal prefix in block-backed output (for example `@alice: ...`) instead of exposing Slack internal mention tokens in the visible body.
- `BasicMarkdown` is rendered with connector-local translation rules:
  - Markdown links `[label](https://...)` are converted to Slack link tokens.
  - `@username` mention tokens are resolved against connector user maps when unambiguous.
  - Mention parsing is skipped inside inline code and fenced code blocks.
  - Emoji shortcodes (for example `:white_check_mark:`) are passed through as shortcodes.
  - Unicode emoji are passed through unchanged.

## Help Rendering Hooks

- Slack connector implements:
  - `FormatHelp(string) string` for line-level Slack-friendly formatting
  - `DefaultHelp() []string` to override no-keyword quick-help lines.
- Built-in help plugin (`builtin-help`) uses these hooks so `help`, `commands`, and `help-all` output remains readable in Slack formatting.

## slack-go v0.17.x Notes

- `api.GetBotInfo` requires `slack.GetBotInfoParameters` (bot + team ID).
- `slackevents.MessageEvent` no longer exposes attachments/timestamps directly; use embedded `Message` payload (`*slack.Msg`) for attachments and timestamps.
- Current library version supports Block Kit send options (`MsgOptionBlocks`) plus `rich_text` / `rich_text_preformatted` block types used by Slack fixed-width output.

## Runtime Lifecycle Notes

- Slack connector runtime state is connector-instance scoped (not package-global).
- Outbound queueing and edited-message dedupe tracking are maintained per connector instance.
- This supports in-process lifecycle operations used by multi-protocol runtime management:
  - `protocol-stop slack`
  - `protocol-start slack`
  - `protocol-restart slack`
  - remove/add `slack` in `SecondaryProtocols` across reloads
