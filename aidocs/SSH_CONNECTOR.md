# SSH Connector (AI Notes)

This document records the intended SSH connector behavior, control flow, and integration points. It is an AI-focused reference aligned with AGENTS.md and startup flow docs.

## Scope

- New connector under `connectors/ssh/`.
- Default connector for local development and IDE mode when no protocol is set.
- Terminal connector remains default for `test-dev` (testbot).

## Startup Integration

- Connector registration via `bot.RegisterConnector("ssh", Initialize)` from `connectors/ssh/static.go`.
- `bot/start.go` should accept `--ssh-port` CLI flag.
- Default selection in `conf/robot.yaml` should use SSH instead of terminal in modes where terminal was implicitly default.
- `nullconn` remains in bootstrapping modes per `aidocs/STARTUP_FLOW.md`.

## ProtocolConfig Shape

`ProtocolConfig` for SSH:

- `ListenHost` (default: `localhost`)
  - Accepts `localhost`, IPv4, IPv6, `all`.
- `ListenPort` (default: `4221`)
- `HostKey` (encrypted private key, optional)
- `ReplayBufferSize` (default: `42`)
- `MaxMsgBytes` (default: `16384`)
- `DefaultChannel` (default: `general`)
- `BotName` (default: `gopherbot`)
- `Channels` (optional list of valid channel names)

`ListenPort` can be overridden by CLI `--ssh-port` and env `GOPHER_SSH_PORT`.

## Connection Model

- One accept goroutine per connector instance.
- One goroutine per client session.
- Client sessions share a global message broadcaster and replay buffer.

## Identity Mapping

- Users are configured in `UserRoster` with `UserID` equal to a full public key line (`ssh-ed25519 AAAA...`).
- Comments are ignored for matching; user name is taken from roster.
- One key per user.
- Bot user is auto-added, using the server host public key line as `UserID`.

## Message Model

### Outgoing formatting

Other user:

```
(09:15:45)@alice/#general(0005): msg
```

Bot message:

```
(09:15:45)=@floyd/#general(0005): msg
```

Thread suffix omitted if not threaded.

Hidden replies:

```
(private/09:15:45)=@floyd/#general(0005): msg
```

### User input

- PTY input; prompt is `@alice/#general -> ` or `@alice/#general(0005) -> `.
- Input echoed normally by PTY.
- On Enter, append `(timestamp)\n`.

### Filters

- Initial filter: `Thread`.
- `A`: all messages.
- `C`: channel messages in current channel (including threads).
- `T`: thread-only when in a thread; otherwise channel-level only.

## Buffer and Size Limits

- Replay buffer size: `ReplayBufferSize` (default 42).
- Each buffered message truncated to 4k; if truncation occurs, append `(WARNING: message truncated to 4k in buffer)` line to the connected client.
- User input size:
  - Accept up to 16k; drop if >16k with `(ERROR: message too long; > 16k - dropped)`.
  - For inputs >4k, buffer truncated 4k copy and emit warning to the sender.
- Bot output size:
  - Send full output to connected clients.
  - Buffer truncated 4k copy.

Hidden messages are never buffered.

## Hidden Messages

- `/botname ...` sends a hidden message to the bot and returns hidden replies only to that user.
- `/ foo` sends nothing to others; emit `(INFO: '/' note to self message not sent to other users)`.
- Hidden replies are prefixed with `private/` in the timestamp segment.

## Commands

- `|c?` list channels; `|c<name>` switch channel
- `|t?` thread help; `|t` toggle thread; `|t<id>` set thread id
- `|j` join last thread seen
- `|f?` or `|f` list filters; `|fA|fC|fT` set filter

No `|u`.

## Paste Handling

- Enable bracketed paste mode on connect; disable on disconnect.
- Treat bracketed paste payload (may include newlines) as a single message.
- For non-bracketed input, line-based input is used.

## Logging

- SSH connector should not call `SetTerminalWriter` (logs go to stdout/robot.log).
- Terminal/test-only admin builtins should be allowed for SSH (treat SSH like terminal/test in `bot/builtins.go`).

## .ssh-connect

Upon successful bind, write `$CWD/.ssh-connect`:

```
BOT_SSH_PORT=127.0.0.1:4221
BOT_SERVER_PUBKEY=ssh-ed25519 AAAA...
```

If host key is auto-generated, log info and include the public key in logs.

## Testbot Exception

`test-dev` mode should still default to terminal connector.
