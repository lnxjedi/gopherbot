# Slack Connector Notes

This file captures Slack connector behavior relevant to routing, private slash commands, and help rendering.

## Source Anchors

- Registration/init: `connectors/slack/static.go`, `connectors/slack/connect.go`
- Incoming message normalization: `connectors/slack/incomingMsgs.go`
- Outgoing formatting + private-command hooks: `connectors/slack/connectorMethods.go`

## Identity Mapping

- Slack connector identity mapping is connector-local in `ProtocolConfig.UserMap` (`username -> Slack user ID`).
- Connector config mapping is treated as canonical when username/ID collisions exist.
- Engine policy checks remain username-based against global `UserRoster`.
- Inbound Slack messages set `ConnectorMessage.ValidatedUser=true` only when the Slack user ID resolves through configured canonical mapping in `ProtocolConfig.UserMap`.
- Slack may still attach a readable `UserName` derived from Slack's own user list for unmapped users, but that username is not trusted for engine security decisions and arrives with `ValidatedUser=false`.
- Inbound worker state preserves both identities:
  - `User` is the canonical Gopherbot username used for policy and plugin-facing identity.
  - `ProtocolUser` preserves the Slack transport identity as a bracketed internal ID (for example `<U0ABC1234>`) when Slack provided one.
- Targeted in-channel sends from engine to connector carry both values:
  - transport ID for routing / live mention rendering
  - canonical username for readable literal prefixes and fallback lookup

## Inbound Message Normalization

- Standard channel/DM messages are passed as canonical `ConnectorMessage` values with:
  - `BotMessage=false`
  - `HiddenMessage=false`
  - `DirectMessage` set from Slack channel type (`IM` vs channel).
- `ValidatedUser=true` only when the inbound Slack user ID matches configured canonical mapping.
- Slack mention tokens are normalized into plain `@username` text before the engine sees them.
- Ordinary messages that contain a bot mention still remain `BotMessage=false`; the engine's bot-name regexes decide whether that normalized text is addressed to the robot.
- Slash command events routed to this app are passed as:
  - `BotMessage=true`
  - `HiddenMessage=true`
  - `MessageText=<slash payload text>`
  - no thread metadata (slash commands are non-threaded inbound).

## Private Command Semantics

- Slack hidden/ephemeral transport support is decided at connector initialization time, not registration time.
- `Initialize(...)` returns `robot.InitializedConnector{Connector, Capabilities}` and only sets `Capabilities.HiddenCommands=true` when slash-command support is explicitly enabled in config.
- Slack protocol config must explicitly set `AcceptSlashCommands: true|false`.
- If `AcceptSlashCommands: true`, `SlashCommand` is required. The connector normalizes either `clu` or `/clu` to the canonical slash form.
- If `AcceptSlashCommands` is omitted, or `SlashCommand` is missing while slash commands are enabled, Slack startup fails with a clear fatal log message so the robot owner knows the config is incomplete.
- Slack slash commands are platform-routed to one bot app, so connector sets `BotMessage=true`.
- Engine private-command policy then treats slash payload as addressed-to-robot without requiring an explicit robot name in text.
- Command still must be explicitly allowed by plugin `AllowedPrivateCommands`, `RequiredPrivateCommands`, or `RequireAllCommandsPrivate`.
- Slack implements `robot.HiddenCommandFormatter`, so engine help/fallback can suggest concrete private slash commands such as `/clu help knock/knock` instead of placeholder `/(bot)` text.
- Engine remains the owner of user-facing denial copy; when a private hidden invocation is matched but addressed incorrectly for the protocol, engine uses Slack's formatter to produce one concrete guidance message.

## Outgoing Format Behavior

- `Raw` keeps legacy Slack-native behavior (including connector-local `@username` handling outside fenced blocks).
- Targeted `Raw` replies add the live mention prefix after Raw formatting so Slack mention tokens are not re-parsed as plain text.
- `Variable` sends a Block Kit `rich_text` block with a plain `rich_text_section` so the visible body preserves the exact message text without relying on Slack markdown parsing.
- Slack does not use zero-width spaces for this visible `Variable` body; visual rendering and normal copy/paste fidelity are preserved for the block-backed text.
- `Fixed` sends a Block Kit `rich_text` block using `rich_text_preformatted` so fixed-width output renders as native Slack preformatted content instead of literal triple-backtick fences.
- Slack does not use zero-width spaces for this visible `Fixed` body either; visual rendering and normal copy/paste fidelity are preserved for the block-backed preformatted text.
- Block-backed `Variable` / `Fixed` sends still include top-level fallback text for notifications/accessibility and legacy RTM fallback behavior.
- Some fallback/legacy Slack text paths still use connector-local soft-hyphen padding to avoid Slack re-parsing, but that is distinct from the visible block-backed `Variable` / `Fixed` body and is not a Google Chat-style ZWSP literalization strategy.
- Long `Variable` / `Fixed` sends currently use a conservative application chunk size of 2,800 characters per chunk rather than Slack's theoretical maximum for these block types.
- Targeted user-in-channel sends use a readable literal prefix in block-backed output (for example `@alice: ...`) instead of exposing Slack internal mention tokens in the visible body.
- `BasicMarkdown` is sent through Slack's native `markdown_text` field for normal Web API sends.
  - The connector preserves the original BasicMarkdown syntax instead of translating links/emphasis/code into legacy Slack `mrkdwn`.
  - `@username` mention tokens are still resolved against connector user maps when unambiguous and rewritten to Slack user mention tokens (`<@U...>`).
  - Targeted `BasicMarkdown` replies still add their live mention prefix before send.
  - Mention parsing is skipped inside inline code and fenced code blocks.
  - Escapes, shortcode emoji, Unicode emoji, quotes, lists, and markdown links are passed through in their original BasicMarkdown form.
  - Long `BasicMarkdown` sends currently chunk at a conservative application limit of 11,500 characters, below Slack's documented `markdown_text` maximum of 12,000 characters.
  - If Web API send falls back to legacy RTM send, the connector still derives a legacy-formatted fallback text for that chunk.

## Help Rendering Hooks

- Slack connector implements `FormatHiddenCommand(string) string` via `robot.HiddenCommandFormatter`.
- `FormatHiddenCommand(...)` uses the configured slash command, not Slack bot username, so help/fallback suggestions reflect the real slash command surface (`/clu ...`) instead of app/bot identity values such as `clu_gopherbot`.
- `DefaultHelp()` now returns no override so the engine can keep protocol-agnostic quick-help text instead of hardcoding Slack slash syntax.
- Built-in help and fallback are rendered in engine-owned `BasicMarkdown`; the connector still contributes only protocol-specific hidden-command guidance where needed.

## slack-go v0.21.x Notes

- `api.GetBotInfo` requires `slack.GetBotInfoParameters` (bot + team ID).
- `slackevents.MessageEvent` no longer exposes attachments/timestamps directly; use embedded `Message` payload (`*slack.Msg`) for attachments and timestamps.
- Current library version supports Block Kit send options (`MsgOptionBlocks`) plus `rich_text` / `rich_text_preformatted` block types used by Slack fixed-width output.
- Current library version also exposes `MsgOptionMarkdownText`, which the connector now uses for `BasicMarkdown` sends.
- `MaxMessageSplit` remains the Slack connector's operator-facing cap on how many chunks/messages one long outbound send may emit before truncation.

## Runtime Lifecycle Notes

- Slack connector runtime state is connector-instance scoped (not package-global).
- Outbound queueing and edited-message dedupe tracking are maintained per connector instance.
- During normal engine reload, Slack `Reload()` refreshes connector-local `ProtocolConfig.UserMap` without reconnecting to Slack.
- The reload path normalizes the new map first, then swaps the configured identity overlay and related username/ID lookup entries under the connector lock so inbound validation and outbound mention lookup see a complete old or complete new map.
- Other transport lifecycle settings that require reconnecting remain controlled by protocol restart/startup rather than `Reload()`.
- This supports in-process lifecycle operations used by multi-protocol runtime management:
  - `protocol-stop slack`
  - `protocol-start slack`
  - `protocol-restart slack`
  - remove/add `slack` in `SecondaryProtocols` across reloads
