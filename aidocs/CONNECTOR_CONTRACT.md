# Connector Contract

This file is the first-draft cross-protocol contract for how connectors hand inbound messages to the engine and what the engine may assume in return.

It is intentionally focused on behavior that must stay consistent across protocols. Connector-specific docs still own transport-local details.

## Source Anchors

- Shared connector types: `robot/connector_defs.go`
- Engine inbound routing: `bot/handler.go`
- Slack reference connector: `connectors/slack/incomingMsgs.go`, `connectors/slack/messages.go`
- Google Chat reference connector: `connectors/googlechat/connector.go`, `connectors/googlechat/ambient.go`

## Contract Goals

- Keep transport cleanup in connectors.
- Keep command-addressing decisions in the engine whenever possible.
- Keep identity/authorization decisions username-based in engine flows.
- Preserve cross-protocol behavioral consistency for ordinary user messages.

## Inbound Message Rules

### 1. Connectors normalize transport markup into readable text

Connectors should remove or rewrite protocol-local message markup before handing text to the engine.

Examples:
- Slack mention tokens like `<@U123>` become plain `@alice`.
- Google Chat mention spans like `@Bishop Gopherbot` become plain `@bishop` when the mention target is the bot's canonical name.

The resulting `ConnectorMessage.MessageText` should resemble the human-visible sentence closely enough that engine regex matching behaves consistently across protocols.

### 2. Mention presence alone does not imply bot-addressed command mode

Normal user messages that merely contain a bot mention must not be forced into `BotMessage=true`.

Examples:
- `Did you see what @bishop did?` remains a normal ambient message.
- `@bishop ping` remains a normal message text that the engine may recognize as addressed to the bot through its existing name/alias regexes.

This preserves the engine as the owner of name-based command detection.

### 3. `BotMessage=true` is reserved for transport-native directed interactions

Connectors should set `BotMessage=true` only when the transport has already made the message unambiguously bot-directed.

Examples:
- slash commands routed directly to the bot
- app-command events invoked through protocol-native command UI

Transport-visible mentions in ordinary message text are not sufficient by themselves.

### 4. `HiddenMessage=true` is reserved for transport-private invocation paths

Connectors should set `HiddenMessage=true` only when the transport guarantees the invocation is private to the user and the bot.

Examples:
- Slack slash commands
- Google Chat slash commands

The engine remains the owner of hidden-command policy and user-facing denial/help behavior.

### 5. `DirectMessage` is connector-authoritative

Connectors are the sole authority for whether an inbound message is a DM/private conversation.

The engine may rely on:
- `DirectMessage=true` for DM-only command eligibility
- `DirectMessage=false` for channel/space semantics

The engine must not infer or overwrite this after connector normalization.

## Identity Rules

- Connectors may use transport-local internal user IDs for routing and live mentions.
- Connectors should map transport identity to canonical Gopherbot username deterministically through connector config or authoritative transport data.
- Engine policy remains username-authoritative.
- A connector must not invent cross-protocol identity equivalence heuristically.

## Threading Rules

- Connectors may preserve protocol-local thread metadata in `ThreadID` and `ThreadedMessage`.
- Connector-local default-thread reply behavior is allowed, but it must not redefine engine-wide send semantics.

## Hidden Command Formatting

- If a protocol has a native hidden/private command surface, the connector should implement `robot.HiddenCommandFormatter` so engine help/fallback can show protocol-real examples.
- The engine owns wording and policy; the connector only supplies transport-specific rendering of the command surface.

## Current Reference Behavior

- Slack follows this contract by rewriting mentions into plain `@username` text and leaving ordinary messages as `BotMessage=false`.
- Google Chat should follow the same model by rewriting mention spans into plain `@username` text and reserving `BotMessage=true` for slash/app-command-style interactions.
