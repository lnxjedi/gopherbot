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
Connectors must not enforce plugin channel restrictions or
`RestrictPrivateChannels`; they only provide accurate message context,
transport identity, and hidden-command capability/formatting.

For outbound replies in that same hidden context, connectors may recover the original invoking user from the inbound hidden message so transport-private reply semantics are preserved even when engine code uses a channel/thread-style send helper.

### 5. `DirectMessage` is connector-authoritative

Connectors are the sole authority for whether an inbound message is a DM/private conversation.

The engine may rely on:
- `DirectMessage=true` for DM-only command eligibility
- `DirectMessage=false` for channel/space semantics

The engine must not infer or overwrite this after connector normalization.

## Identity Rules

- Connectors may use transport-local internal user IDs for routing and live mentions.
- Connectors should map transport identity to canonical Gopherbot username deterministically through connector config or authoritative transport data.
- `ConnectorMessage.ValidatedUser=true` means the connector can vouch that this specific `UserID` maps to this canonical `UserName`.
- Connectors must not set `ValidatedUser=true` for a guessed, display-name-derived, or heuristic username.
- Engine policy remains username-authoritative.
- A connector must not invent cross-protocol identity equivalence heuristically.

### Validated User Contract

- `ValidatedUser=true` is the trust boundary for inbound security decisions.
- `ValidatedUser=false` is allowed for ordinary ambient traffic when the connector can observe a transport user but cannot yet vouch for the canonical Gopherbot username.
- Engine pre-pipeline user filtering may reject a message even when `UserName` is present, if `ValidatedUser` is false.
- The intended pattern is:
  - local/authenticated connectors like SSH/terminal/test set `ValidatedUser=true` for their configured users
  - Slack/Google Chat set `ValidatedUser=true` only when the transport ID resolves through connector-local canonical mapping such as `ProtocolConfig.UserMap`
  - unmapped Slack/Google Chat users may still arrive with `UserName` text for human readability, but with `ValidatedUser=false`

## Reload Rules

- `robot.Connector.Reload() error` is called by the engine for each active connector after a successful normal configuration reload has loaded the new protocol config and reconciled active secondary connectors.
- Reload is for connector-local runtime configuration that can be applied without reconnecting the transport. Current identity examples:
  - Slack `ProtocolConfig.UserMap`
  - Google Chat `ProtocolConfig.UserMap`
  - SSH `ProtocolConfig.UserKeys`
- Connector reload implementations must parse and normalize new config before mutating live state.
- Connector reload implementations must apply live state changes atomically under connector-owned locks so concurrent readers see either the old complete mapping or the new complete mapping.
- Connector reload must not bypass engine policy. It only changes connector-local mapping/lookup state; authorization and command availability remain engine-owned and username-authoritative.
- A reload failure in one connector is logged by the engine and does not stop other active connectors from attempting reload.

## Threading Rules

- Connectors may preserve protocol-local thread metadata in `ThreadID` and `ThreadedMessage`.
- Connector-local default-thread reply behavior is allowed, but it must not redefine engine-wide send semantics.

## Hidden Command Formatting

- If a protocol has a native hidden/private command surface, the connector should implement `robot.HiddenCommandFormatter` so engine help/fallback can show protocol-real examples.
- The engine owns wording and policy; the connector only supplies transport-specific rendering of the command surface.

## Current Reference Behavior

- Slack follows this contract by rewriting mentions into plain `@username` text and leaving ordinary messages as `BotMessage=false`.
- Google Chat should follow the same model by rewriting mention spans into plain `@username` text and reserving `BotMessage=true` for slash/app-command-style interactions.
