# SSH Connector (AI Notes)

This document records the intended SSH connector behavior, control flow, and integration points. It is an AI-focused reference aligned with AGENTS.md and startup flow docs.

## Scope

- New connector under `connectors/ssh/`.
- Default connector for local development and IDE mode when no protocol is set.
- Terminal connector remains default for `test-dev` (testbot).

## Source Layout

- `connectors/ssh/connector.go`: connector interface surface, message routing, buffer logic.
- `connectors/ssh/server.go`: SSH listener, auth, and session lifecycle.
- `connectors/ssh/readline.go`: readline-based input/output, prompts, formatting.
- `golib/readline/`: local fork of `github.com/chzyer/readline` (see Bracketed Paste Support below).

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
- `Color` (default: `true` in stock config)
- `ColorScheme` (ANSI 256 color map; keys: `prompt`, `timestamp`, `bot`, `user`, `system`, `info`, `warning`, `error`, `private`)

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

Direct message (channel view, inbound):

```
(09:15:45)from:@bob: msg
```

Direct message (replay, outbound):

```
(09:15:45)to:@bob: msg
```

Direct message (DM view, inbound):

```
(09:15:45)@bob: msg
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

Color output uses ANSI 256 sequences. User input remains uncolored; prompts, bot/user messages, and system lines are colorized via `ColorScheme`. Message headers (timestamp/user/channel) are colorized per segment.

### User input

- PTY input; prompt is `@alice/#general -> ` or `@alice/#general(0005) -> `.
- Direct-message prompt: `@alice/dm:@bob -> ` (threads disabled in DMs).
- Input echoed normally by PTY.
- On Enter, if the timestamp fits on the same line, append ` (HH:MM:SS)` via the readline painter; otherwise no inline timestamp is shown.
- Input line editing uses `github.com/chzyer/readline`; history is per-session only (no persistence).

### Filters

- Initial filter: `Channel`.
- `A`: all messages.
- `C`: channel messages in current channel (including threads).
- `T`: thread-only when in a thread; otherwise channel-level plus the first message of each thread (rendered as `(+0005)` to indicate a new thread).
- Direct messages to/from the user are delivered regardless of filter.

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
Direct messages are buffered and replayed only to the sender/recipient.

## Hidden Messages

- `/botname ...` sends a hidden message to the bot and returns hidden replies only to that user.
- `/ foo` sends nothing to others; emit `(INFO: '/' note to self message not sent to other users)`.
- Hidden replies are prefixed with `private/` in the timestamp segment.

## Direct Messages

- `/@user <message>` sends a one-shot DM to a user or the bot.
- `|c` switches to a DM channel with the bot; `|c @user` switches to a DM channel with that user.
- User-to-user DMs are local to the SSH connector and **not** forwarded to the engine.
- In user-to-user DMs, `/` commands are rejected with a warning.
- User names in the roster must be lower-case; uppercase roster entries are rejected with an error log.
- Threads are disabled in DMs.

## Commands

- `|c?` list channels; `|c<name>` switch channel
- `|c` direct message with bot; `|c @user` direct message with user
- `|t?` thread help; `|t` toggle thread; `|t<id>` set thread id
- `|j` join last thread seen
- `|f?` or `|f` list filters; `|fA|fC|fT` set filter
- `|l` list users
- `|?` connector help

No `|u`.

## Paste Handling

- Enable bracketed paste mode on connect; disable on disconnect.
- Bracketed paste payloads are read line-by-line by readline; multi-line paste yields multiple messages.
- For non-bracketed input, line-based input is used.

## Bracketed Paste Support (Readline Fork)

We maintain a local fork of `github.com/chzyer/readline` under `golib/readline/` and use a `go.mod` replace to point at it. The fork adds a callback hook to detect bracketed paste mode transitions:

- `readline.Config` includes `FuncSetPasteMode func(bool)`.
- The terminal escape parser (`golib/readline/terminal.go`) recognizes `ESC [ 200 ~` (paste start) and `ESC [ 201 ~` (paste end), and invokes `FuncSetPasteMode(true/false)`.
- The escape sequences themselves are still filtered out of the input stream.

This hook is wired in `connectors/ssh/readline.go` via `FuncSetPasteMode: client.setPasteActive`, which sets `sshClient.pasteActive`.

### Multiline Input Behavior

Multiline input in the SSH connector now continues when **either**:
- The user ends a line with `\` (manual continuation), or
- Bracketed paste mode is active (paste continuation).

When a multiline input completes:
- A standalone timestamp line is emitted locally.
- The combined text is sent as a single message (joined with `\n`).

### Readline Timestamp Rendering

The SSH connector no longer toggles `UniqueEditLine`. Instead it uses a custom readline `Painter` (`timestampPainter` in `connectors/ssh/readline.go`) to append a timestamp to the rendered input line **only** at Enter time.

- On Enter, `FuncFilterInputRune` calls `setStampIfFits` to decide if the timestamp fits on the same line (no additional wrapping). If it fits, the painter appends ` (HH:MM:SS)` and a refresh is triggered.
- When the line is submitted, the stamp is cleared so normal editing behavior resumes without cursor distortion.
- Multiline continuation (manual `\` or bracketed paste) bypasses the inline timestamp and uses the standalone timestamp line after the multiline block completes.

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
