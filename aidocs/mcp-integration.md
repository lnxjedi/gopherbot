# MCP Integration (AI Dev Tap + Inject)

Status: Draft design notes for the "--aidev" mode. This doc captures the intended behavior, anchor points, and open questions for an AI-facing tap/inject interface.

## Goals

- Allow an AI dev client to observe **all inbound and outbound messages** regardless of connector (Slack, terminal, test, etc.).
- Allow the AI client to **inject** messages as arbitrary users, but with real connector IDs so replies thread correctly (especially Slack).
- Require a **protocol-specific username→userID mapping** for impersonation; injection is denied if the mapping is missing.
- Gate all functionality behind an explicit CLI flag (e.g., `--aidev <secret>`).
- Keep the interface local-only (localhost) and dev-only.

## Non-Goals

- Replacing existing connectors.
- Production-ready auth or remote exposure.
- Modifying behavior when `--aidev` is not set.

## Activation + Guardrails

- Add a CLI flag in `bot/start.go` (func `Start`) to enable the interface.
- The secret is **ephemeral per run** and generated when `--aidev` is set; it is not persisted.
- When `--aidev` is set, gopherbot starts the HTTP listener, logs the exact MCP start command, and **waits for a `start` control action** before starting the connector.
- When the flag is present, start a streamable HTTP server (SSE) on `127.0.0.1` by reusing the existing HTTP listener in `bot/bot_process.go` (same mux as `/json`).
- Require a shared secret header for every request/stream.
  - Header: `X-AIDEV-KEY`
  - Optional envs for MCP start command: `GOPHER_AIDEV_MCP_CONFIG`, `GOPHER_AIDEV_MCP_PROTOCOL`

Anchors:
- CLI flag parsing: `bot/start.go` (func `Start`).
- Existing HTTP handler location (not reused): `bot/bot_process.go` (listener) + `bot/http.go` (handler `ServeHTTP`).

## Current Inbound Message Path (baseline)

- Slack inbound is normalized in `connectors/slack/incomingMsgs.go` (funcs `processMessageSocketMode`, `processMessageRTM`).
- The connector sets `SelfMessage=true` when `userID == s.botUserID` (Slack bot's user ID).
- The message is then passed to `handler.IncomingMessage` in `bot/handler.go`.

The dispatch layer in `bot/dispatch.go` filters behavior based on `Incoming.SelfMessage` for matchers and jobs; if we want an injected message to be treated as a normal user message, we must **clear** `SelfMessage` after rewriting it.

Anchors:
- `bot/handler.go` method `IncomingMessage`.
- `bot/dispatch.go` method `handleMessage` (checks `Incoming.SelfMessage`).

## Loopback Inject Design (connector-agnostic)

### Idea

Injection should round-trip through the active connector so we capture real channel/thread/message IDs. The injected message is sent **by the bot** with a visible prefix and then rewritten on the way back in. This must be done in the engine (not per-connector) so it works across Slack, terminal, and future connectors (Discord, etc.).

### Flow (any connector)

1. AI client posts an inject request to the local aidev server (includes username and userID).
2. Server formats a message like:
   - `(#<nonce> as: <username>) <message>` where `<nonce>` is **7 hex chars**.
3. Server calls the real connector Send* method to send the message.
4. The connector delivers the bot-authored message back into the engine.
5. `handler.IncomingMessage` sees `SelfMessage=true` and the special prefix, then:
   - matches the message to a pending injection (by nonce)
   - rewrites the incoming `ConnectorMessage` to impersonate `<username>` / `<userID>` for the **current protocol**
   - clears `SelfMessage`
   - strips the prefix from `MessageText`
6. The message proceeds through normal `IncomingMessage` handling, and replies thread correctly because the connector provided real message/thread IDs.

### Why rewrite happens in `handler.IncomingMessage`

- We explicitly want connector-agnostic behavior (works for Slack, terminal, future connectors).
- Engine already has access to `SelfMessage` and the final `ConnectorMessage` values.
- A protocol-specific mapping file supplies username→userID data without relying on connector internals.

Anchors:
- Inbound normalization: `connectors/*/incoming` (Slack: `connectors/slack/incomingMsgs.go` funcs `processMessageSocketMode`, `processMessageRTM`).
- Bot ID setup: per connector (Slack: `connectors/slack/connect.go` func `Initialize`, calls `SetBotID`).

### Duplicate/Edit behavior

Slack already ignores edited messages arriving within a 3s window via `ignorewindow` and `lastmsgtime` in `connectors/slack/messages.go`. This only handles `message_changed` edits and does not provide general dedup for loopback messages.

Anchors:
- Dedup window: `connectors/slack/messages.go` (`ignorewindow`, `lastmsgtime`).

## Tap Stream (connector-agnostic)

The AI dev interface should emit both inbound and outbound events, regardless of connector.

- Inbound tap: in `handler.IncomingMessage` after any loopback rewrite but before ignore checks (`bot/handler.go`).
- Outbound tap: wrap `interfaces.SendProtocol*` calls in `bot/send_message.go`, or wrap the connector in `bot/bot_process.go` when `setConnector` is called.

Event payload should include:
- direction: `inbound` | `outbound`
- protocol
- user/channel names + IDs
- thread/message IDs
- flags (direct, hidden, botMessage, selfMessage)
- text

### SSE endpoint

- `GET /aidev/stream`
- Requires `X-AIDEV-KEY` header matching `--aidev` secret.
- Streams JSON events as SSE `data:` lines.

Example event (shape only):

```
{
  "Direction": "inbound",
  "Protocol": "slack",
  "UserName": "david",
  "UserID": "U0123ABCDE",
  "ChannelName": "general",
  "ChannelID": "C0ABCDEF",
  "ThreadID": "1700000000.000100",
  "MessageID": "1700000000.000100",
  "SelfMessage": false,
  "BotMessage": false,
  "Hidden": false,
  "Direct": false,
  "Text": "hello"
}
```

## Injection API (high-level + mapping)

Injection requests MUST include the userID from the MCP mapping. The server should refuse injections without a mapping entry. The engine **trusts** the shim-provided userID (no engine-side cross-check).

```
{
  "user": "david",
  "user_id": "U0123ABCDE",
  "channel": "general",
  "thread": "<optional>",
  "message": "hello",
  "direct": false
}
```

The mapping file lives with the MCP shim and supplies user_id; the engine uses it only for loopback rewrite.

### Inject endpoint

- `POST /aidev/inject`
- Requires `X-AIDEV-KEY` header matching `--aidev` secret.
- Returns `202 Accepted` on success, `400/401` otherwise.
- Returns `409 Conflict` if the robot has not completed post-connect initialization.

### Control endpoint

- `POST /aidev/control`
- Requires `X-AIDEV-KEY` header matching `--aidev` secret.
- Payload: `{ "action": "hello" | "ready" | "start" | "exit" | "force_exit" | "stack_dump" }`
- `exit` triggers graceful shutdown (`stop()`).
- `force_exit` triggers a SIGUSR1 stack dump + panic (see `bot/signal.go`).
- `stack_dump` logs a runtime stack dump without exiting.
- `hello` is used by `gopherbot-mcp` to confirm it can reach the control endpoint.
- `ready` is sent after the MCP connects to the SSE stream.
- `start` tells gopherbot to continue startup (connector + post-connect config).

## Pending Injection Tracking

A minimal pending queue is needed to correlate loopback messages when multiple injections are in flight.

Suggested key:
- nonce (hex string) embedded in the prefix.

Suggested structure:
- map[nonce]pendingInjection{user, userID, channel, thread, sentAt}
- TTL cleanup (short window, e.g., 30s)

## Prefix Format

Current candidate:
- `(#<nonce> as: <username>) <message>`

Constraints:
- Visible in Slack for dev transparency.
- Unlikely to collide with normal messages.
- Must parse reliably (strict prefix parser).

## Open Questions

- Should the prefix include channel/thread hints, or only user + nonce?
- Should loopback rewriting be allowed only when `--aidev` is set? (Yes.)
- Should there be a raw injection mode for tests, or only high-level payloads?
- Should injected messages bypass `IgnoreUnlistedUsers` in `bot/handler.go`? (No: evaluate after rewrite; log when ignored.)

## Related Files (entrypoints)

- `bot/start.go` (func `Start`) — CLI flags.
- `bot/handler.go` (func `IncomingMessage`) — inbound tap.
- `bot/send_message.go` — outbound tap.
- `cmd/gopherbot-mcp/main.go` — MCP bridge, hello handshake.
- `connectors/slack/incomingMsgs.go` — SelfMessage flagging (Slack).
- `connectors/slack/connect.go` — bot ID init (Slack).
- `connectors/slack/messages.go` — edit dedup window.
