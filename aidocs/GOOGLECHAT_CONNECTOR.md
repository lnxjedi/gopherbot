# Google Chat Connector Notes

This file captures Google Chat connector behavior relevant to routing, hidden commands, identity mapping, ambient Workspace Events subscriptions, threading, and outgoing message formatting.

## Source Anchors

- Registration/init: `connectors/googlechat/static.go`, `connectors/googlechat/connect.go`
- Incoming event normalization + send behavior: `connectors/googlechat/connector.go`
- Ambient Workspace Events subscription lifecycle + CloudEvent normalization: `connectors/googlechat/ambient.go`, `connectors/googlechat/workspaceevents.go`
- BasicMarkdown rendering: `connectors/googlechat/basic_markdown.go`
- Sample config: `conf/protocols/googlechat.yaml.sample`
- Shared Google credential loading: `internal/gcloud/credentials.go`

## Transport Model

- Google Chat uses the Chat API for outbound messages and a Pub/Sub pull subscription for inbound events.
- Connector initialization loads encrypted service-account credentials through `Handler.ReadEncryptedFile(...)` and `internal/gcloud`.
- Runtime receive uses one Pub/Sub goroutine with one outstanding message at a time so connector-local event handling remains serialized.
- At `Info` log level, the connector logs concise summaries of inbound Pub/Sub deliveries so operators can distinguish Chat interaction events from Workspace Events deliveries while debugging Google-side configuration.
- The connector is text-only in v1. It does not expose cards, dialogs, or other Google Chat-specific UI surfaces through the shared connector contract.
- The same Pub/Sub subscription receives both normal Chat interaction events and Google Workspace Events CloudEvents when ambient-space subscriptions are enabled.

## Ambient Message Support

- `ProtocolConfig.AmbientMessages` enables connector-managed Google Workspace Events subscriptions.
- When enabled, the connector:
  - lists non-DM spaces the app is already a member of
  - creates or renews a per-space Workspace Events subscription targeting the shared Pub/Sub topic
  - refreshes those subscriptions periodically before expiration
  - creates a new subscription when Chat reports `ADDED_TO_SPACE`
  - deletes the subscription when Chat reports `REMOVED_FROM_SPACE`
- Ambient delivery currently subscribes to Chat message-created events and normalizes them into the same `ConnectorMessage` flow as interaction events.
- This path requires administrator-approved `chat.app.*` scopes and the Google Workspace Events API. Terraform remains project-level only; the connector owns per-space subscription lifecycle.
- Because Chat app ambient subscriptions currently use payload data, subscription TTL is short-lived and the connector renews them automatically.

## Identity Mapping

- Google Chat connector identity mapping is connector-local in `ProtocolConfig.UserMap` (`username -> users/{id}`).
- Engine policy checks remain username-based against global `UserRoster`.
- Inbound interaction events carry the Google Chat user resource name (for example `users/12345678901234567890`) as `ConnectorMessage.UserID`.
- If the resource name exists in `ProtocolConfig.UserMap`, the connector sets `ConnectorMessage.UserName` to the canonical Gopherbot username.
- If a human user is not mapped, the connector leaves canonical username unset and logs a one-time warning including:
  - display name
  - email (when provided by Chat)
  - Google Chat user resource name
- This warning is the intended operator-discovery path for finding the internal Google Chat user ID to add to `UserMap`.

## Inbound Message Normalization

- Google Chat interaction events that produce bot input are normalized as `Protocol: "googlechat"`.
- Google Workspace Events message-created CloudEvents are also normalized as `Protocol: "googlechat"`.
- Mention spans are rewritten into Slack-style plain-text mentions before the engine sees them.
  - bot mentions become `@<bot username>` using the robot's canonical bot username
  - mapped human mentions become `@<canonical username>`
  - unmapped mentions are left in their human-visible text form
- For `MESSAGE` and supported `APP_COMMAND` events:
  - `BotMessage=true` only for slash-command-style and app-command-style interactions that Google already routes directly to the app
  - ordinary text messages, including those that mention the app, use `BotMessage=false`
  - ordinary message text is normalized from `message.text`, not collapsed to `message.argumentText`, so engine bot-name regexes remain the authority for deciding whether the message is addressed to the bot
- For ambient Workspace Events message-created events:
  - `BotMessage=false`
  - `HiddenMessage=false`
  - text is normalized from `message.text` using the same mention-rewrite rules as interaction events
- `DirectMessage=true` when `space.spaceType == DIRECT_MESSAGE`.
- For direct messages, `ChannelID`/`ChannelName` are left empty so engine DM behavior continues to use `SendProtocolUserMessage(...)`.
- For space messages:
  - `ChannelID` is the Chat space resource name (`spaces/{space}`)
  - `ChannelName` is the display name when Chat provides one
- Thread normalization:
  - `ThreadID` is the Chat thread resource name when present
  - `ThreadedMessage=true` only when the incoming Chat message is already a reply in a thread
  - root/top-level threaded-space messages still carry `ThreadID` for connector-local default threading decisions
- The connector keeps a short-lived seen-message cache so the same Chat message is not processed twice if both an interaction event and a Workspace Events delivery arrive for it.

## Hidden Command Semantics

- Hidden-command support is enabled when `ProtocolConfig.SlashCommand` is configured.
- Google Chat slash commands are private to the invoking user and the Chat app, so connector maps slash-command events to:
  - `HiddenMessage=true`
  - `BotMessage=true`
- Connector implements `robot.HiddenCommandFormatter`.
- Help/fallback rendering uses the configured slash command name, for example `/bishop help ping`.
- When replying to a hidden slash command:
  - if reply stays in the same user + same space context, connector uses `privateMessageViewer` so only that user sees the response
  - if target user or space changes, connector drops hidden/private treatment and sends a normal visible message instead

## ThreadResponses

- `ProtocolConfig.ThreadResponses` defaults to `true`.
- When enabled, if an outbound send does not specify a thread explicitly and the send stays in the same originating Google Chat user/space context, the connector reuses the inbound `ThreadID`.
- Practical effect:
  - normal `Say()` / `Reply()` behavior in Google Chat tends to stay in the originating thread
  - explicit thread methods still win if engine/plugin code supplies a thread ID
- This is connector-local behavior only; engine-wide send semantics are unchanged.

## Outgoing Message Formatting

The connector is text-only in v1 and sends through the Google Chat `Message.text` field.

- `BasicMarkdown`:
  - converted to Google Chat text-message syntax
  - `**bold**` -> Chat bold
  - `*italic*` -> Chat italic
  - fenced code, inline code, block quotes, and single-level unordered lists are preserved in Chat-compatible text form
  - `[label](url)` -> Chat hyperlink syntax
  - `@username` -> `<users/{id}>` only when `UserMap` resolves unambiguously; otherwise literal `@username` is preserved
- `Variable`:
  - sent as normal Chat text without protocol-specific decoration
- `Fixed`:
  - sent as a fenced monospace block
- `Raw`:
  - treated as literal-ish text passthrough
  - connector does not attempt to interpret protocol-specific Slack-style raw formatting

## Directed Replies

- Google Chat has no separate native “user-in-channel” send primitive in the shared connector contract.
- For visible directed replies in a space, the connector prefixes a real Chat mention (`<users/{id}>: `) and posts the message into the target space/thread.
- For hidden slash-command replies in the original context, the connector prefers `privateMessageViewer` over a visible mention prefix.

## Limitations

- No card/dialog/widget support through the connector contract in v1.
- `JoinChannel(...)` is not implemented for Google Chat spaces and returns `FailedChannelJoin`.
- Arbitrary channel lookup by plain display name is best-effort from connector-observed spaces only; the authoritative route is the Chat space resource name (`spaces/{space}`).
- Ambient message capture currently only consumes message-created events. Message edits/deletes are not routed into the engine.
