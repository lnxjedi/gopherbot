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

- Connector registration via `robot.RegisterConnector("ssh", Initialize)` from `connectors/ssh/static.go`.
- `connectors/ssh/connector.go` `Initialize(...)` returns `robot.InitializedConnector{Connector, Capabilities}` and marks SSH hidden-command support there.
- `bot/start.go` should accept `--ssh-port` CLI flag.
- Default selection in `conf/robot.yaml` should use SSH instead of terminal in modes where terminal was implicitly default.
- `nullconn` remains in bootstrapping modes per `aidocs/STARTUP_FLOW.md`.

## ProtocolConfig Shape

`ProtocolConfig` for SSH:

- `ListenHost` (default: `localhost`)
  - Accepts `localhost`, IPv4, IPv6, `all`.
- `ListenPort` (default: `4221`)
  - Treated as the base port for startup probing.
  - On startup, SSH tries `ListenPort` first, then increments one port at a time up to `ListenPort+7`.
  - The first available port in that 8-port window is used; if all are in use, connector startup fails.
  - When probing skips in-use ports, startup logs the skip count.
  - `.ssh-connect` always records the actual selected port.
- `HostKey` (encrypted private key, optional)
- `HearSelf` (default recommended: `true` for SSH)  
  - when true, connector-injected bot-authored events (such as join announcements)
    are forwarded back through `IncomingMessage` as `SelfMessage=true`
- `ReplayBufferSize` (default: `42`)
- `MaxMsgBytes` (default: `16384`)
- `DefaultChannel` (default: `general`)
- `Channels` (optional list of valid channel names)
- `Color` (default: `true` in stock config)
- `ColorScheme` (ANSI 256 color map; keys: `prompt`, `timestamp`, `bot`, `user`, `system`, `info`, `warning`, `error`, `private`, `inlinecode`, `codeblock`)

`ListenPort` can be overridden by CLI `--ssh-port` and env `GOPHER_SSH_PORT`.

SSH no longer takes a protocol-local `BotName`. During `Initialize(...)`, it
uses `Handler.GetBotInfo().UserName` as the canonical robot name for
name-addressed hidden input, hidden-help rendering, and local bot labeling. If
`BotInfo.UserName` is empty, SSH falls back to `gopherbot`.

## Connection Model

- One accept goroutine per connector instance.
- One goroutine per client session.
- Client sessions share a global message broadcaster and replay buffer.

## Prompt Behavior

- Prompt* waits from ssh sessions use the engine's extended interactive timeout (`42m`) when the caller is a compiled Go or interpreter-backed (`.go`, `.lua`, `.js`) task.
- Prompt* waits are canceled immediately during robot shutdown so pending interactive prompts do not delay stop/restart.

## Identity Mapping

- SSH connector identity mapping is connector-local in `ProtocolConfig`.
- `ProtocolConfig.UserKeys` is a list of entries with:
  - `UserName`
  - `PublicKeys` (`[]string`)
- This list shape is intentional: config load merges maps by key, while list values are replaced unless explicitly using `Append*` keys. Using a list prevents default SSH users from merging into custom robot identity config.
- Custom robots that want to clear installed defaults should set `ProtocolConfig.UserKeys: []` (empty list) before adding their own entries.
- `UserRoster` remains the global user directory (email/name/phone/etc.) and policy membership list.
- Comments are ignored for matching; user name is taken from mapping/roster.
- Multiple keys per user are supported.
- Bot user is auto-added, using the server host public key line as `UserID`.
- Engine outbound user-targeting is username-based; SSH resolves connector-local user IDs internally.
- Inbound SSH messages are always emitted with `ConnectorMessage.ValidatedUser=true` for authenticated configured users because the connector authenticated the presented public key and matched it to the configured canonical username.
- During normal engine reload, SSH `Reload()` refreshes `ProtocolConfig.UserKeys` without restarting the listener.
- The reload path builds the new key/name/ID indexes first, then swaps all three maps plus the stored `UserKeys` config under the connector lock. New authentication attempts use the new complete key map.
- Existing sessions authenticated with keys no longer present in `UserKeys` are closed after the atomic map swap.
- Listener settings such as `ListenHost`, `ListenPort`, and `HostKey` remain startup/restart concerns rather than live reload behavior.

## Message Model

### Presence join announcement

- After an authenticated SSH client is added, the connector emits a bot-authored
  channel message in the client's current channel:
  - `@<username> has joined #<channel>`
- This message is appended to the SSH replay buffer in normal sequence order.
- Live broadcast excludes the joining user's own active session(s); other users
  in matching views/filters see the announcement.
- The connector also forwards a canonical SSH inbound message for the same
  event through `handler.IncomingMessage(...)` when `HearSelf` is enabled.
  The forwarded event is marked `SelfMessage=true`.

### Self-hear note

- SSH now supports `ProtocolConfig.HearSelf` for connector-originated
  bot-authored events (for example join announcements).
- Engine routing behavior for `SelfMessage=true` is important:
  - plugin command/message matching is skipped
  - job triggers still run (see `aidocs/PIPELINE_LIFECYCLE.md`)

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

Message format behavior:

- `Raw` / `Variable`: preserved as-is (same connector behavior as before).
- `Fixed`: preserved as-is; when a fixed message has multiple lines, SSH displays
  the header on its own line and starts the fixed body at column zero without
  normal message wrapping so table columns remain aligned.
- `BasicMarkdown`: rendered to plain-text-safe output for SSH:
  - when `Color` is enabled, bold and italics use ANSI SGR in live SSH output; otherwise markers are removed
  - when `Color` is enabled, inline code uses the `inlinecode` color and fenced code blocks use the `codeblock` color in live SSH output
  - inline code/backticks removed, keeping inner text
  - fenced code fences removed (including optional language hint), preserving code lines
  - markdown links degraded to `label (https://...)`
  - escaped literals (for example `\*`, ``\` ``, `\@`) unescaped
  - mentions pass through as literal text
  - core emoji shortcodes (for example `:white_check_mark:`, `:rocket:`) are expanded to Unicode outside inline/fenced code; unknown shortcodes remain literal

### Rendering Pipeline

- SSH stores a plain-text `text` payload in the replay buffer and `GetMessages(...)` results.
- For `BasicMarkdown`, the connector also preserves the original markdown source separately on the buffered event (`basicMarkdownSource`) when the message is not truncated.
- Live SSH display re-renders from `basicMarkdownSource` when `Color` is enabled, so terminal-only styling stays out of the canonical buffered/API text.
- Styled output is wrapped after ANSI sequences are added; the shared wrapper in `robot/util/wrap.go` ignores ANSI escape width so bold/italic/code colors do not break line wrapping.
- If a buffered message is truncated to the 4k replay limit, SSH clears the preserved markdown source so replay/API consumers do not receive partial terminal styling state.

### User input

- PTY input; prompt is `@alice/#general -> ` or `@alice/#general(0005) -> `.
- Direct-message prompt: `@alice/dm:@bob -> ` (threads disabled in DMs).
- Input echoed normally by PTY.
- On Enter, append an inline ` (HH:MM:SS)` timestamp via the readline submit hook; if it would split across the line boundary, pad so `(` starts at the next line. The inline timestamp is colorized using the `timestamp` color.
- Input line editing uses `github.com/chzyer/readline`; history is per-session only (no persistence).

### Filters

- Initial filter: `Channel`.
- `A`: all messages.
- `C`: channel messages in current channel (including threads).
- `T`: thread-only when in a thread; otherwise channel-level plus the first message of each thread (rendered as `(+0005)` to indicate a new thread).
- Direct messages to/from the user are delivered regardless of filter.

## Buffer and Size Limits

- Replay buffer size: `ReplayBufferSize` (default 42).
- Buffered messages include a monotonic connector cursor (`seq`) for polling/catch-up APIs.
- Each buffered message truncated to 4k; if truncation occurs, append `(WARNING: message truncated to 4k in buffer)` line to the connected client.
- User input size:
  - Accept up to 16k; drop if >16k with `(ERROR: message too long; > 16k - dropped)`.
  - For inputs >4k, buffer truncated 4k copy and emit warning to the sender.
- Bot output size:
  - Send full output to connected clients.
  - Buffer truncated 4k copy.

Hidden bot replies are buffered with viewer-scoped visibility (only the target user can retrieve/replay them).
Direct messages are buffered and replayed only to the sender/recipient.

## Hidden Messages

- `/botname ...` sends a hidden message addressed by robot name and returns hidden replies only to that user.
- `/...` (without bot name) is still hidden from other SSH users, but is not considered a name-addressed hidden command by engine policy.
- `/ foo` sends nothing to others; emit `(INFO: '/' note to self message not sent to other users)`.
- Incoming SSH hidden messages therefore split into two cases:
  - `HiddenMessage=true` for any slash-prefixed hidden/private message
  - a robot-addressed command payload only for `/<botname> ...`, which SSH normalizes to `<BotInfo.UserName> ...` before calling `IncomingMessage(...)`
- Hidden replies are prefixed with `private/` in the timestamp segment.
- SSH advertises hidden-command support from `Initialize(...)` through `robot.InitializedConnector.Capabilities.HiddenCommands`.
- SSH implements `robot.HiddenCommandFormatter`, so built-in help and metadata can render concrete hidden examples such as `/clu help ping` and `/clu ping`.
- The same formatter is used by engine-owned denial copy when a hidden command is matched but not addressed with SSH hidden syntax.
- Engine-side hidden policy still applies:
  - command must be listed in plugin `AllowedHiddenCommands`
  - and hidden message must be robot-addressed (`/<botname> ...` for SSH, or connector-routed `BotMessage=true` in protocols like Slack slash commands).

## AI-Dev Injection and Retrieval

When the connector is running under `--aidev`, SSH exposes optional connector API capabilities used by engine endpoints:

- `InjectMessage(...)` injects an inbound SSH message as a mapped roster user, then routes through `handler.IncomingMessage` (engine auth/business logic still applies).
- `GetMessages(...)` returns viewer-visible messages with cursor semantics:
  - `all=true`: visible buffer snapshot (oldest -> newest)
  - `after_cursor=N`: messages with `seq > N`
  - if none are available, optional long-poll wait up to `timeout_ms`

Visibility rules for retrieval:

- public channel messages: visible to all viewers
- direct messages: visible only to participants
- hidden bot replies: visible only to the scoped hidden recipient

Recommended MCP usage pattern for fresh/unknown robots:

- after `start_robot`, call `send_message` in `#general` with `help` (or `info`) and then poll `get_messages`
- do not assume the bot name matches directory name (for example `../bishop` may still identify as `floyd`)
- extract the active bot name/alias from the help/intro text, then address follow-up commands accordingly
- continue with cursor polling (`after_cursor`, default `timeout_ms=1400`) to capture multi-message workflows

Conversation notes from live MCP interaction:

- command replies may be multi-message and staggered in time (poll until quiet, not just one response)
- `send_message` returns injected thread/message metadata, but bot replies are plugin-dependent and may be unthreaded
- if a natural-language addressed message returns `No command matched ...; try ';help'`, pivot to `;help <keyword>` to discover accepted syntax
- plain non-addressed chat in channel can be used for commentary/rating when no matching command exists
- a `get_messages` response with `TimedOut=true` and unchanged cursor is the expected "no new messages yet" signal

## Direct Messages

- `/@user <message>` sends a one-shot DM to a user or the bot.
- `|c` switches to a DM channel with the bot; `|c @user` switches to a DM channel with that user.
- DMs with the bot are forwarded to engine as canonical `ConnectorMessage` events (`DirectMessage=true`).
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

## Readline Fork Extensions

We maintain a local fork of `github.com/chzyer/readline` under `golib/readline/` and use a `go.mod` replace to point at it. The fork adds a callback hook to detect bracketed paste mode transitions:

- `readline.Config` includes `FuncSetPasteMode func(bool)`.
- The terminal escape parser (`golib/readline/terminal.go`) recognizes `ESC [ 200 ~` (paste start) and `ESC [ 201 ~` (paste end), and invokes `FuncSetPasteMode(true/false)`.
- The escape sequences themselves are still filtered out of the input stream.

We also add a submission hook to support inline timestamps without violating readline's width accounting:

- `readline.Config` includes `FuncBeforeSubmit func(line []rune) (suffix []rune, strip int)`.
- The hook runs on Enter before the line is finalized, allowing a transient suffix to be appended for display while stripping it from the returned line and history.

This hook is wired in `connectors/ssh/readline.go` via `FuncBeforeSubmit`, which appends the timestamp suffix and strips it from the submitted line/history.

### Multiline Input Behavior

Multiline input in the SSH connector now continues when **either**:
- The user ends a line with `\` (manual continuation), or
- Bracketed paste mode is active (paste continuation).

When a multiline input completes:
- A standalone timestamp line is emitted locally.
- The combined text is sent as a single message (joined with `\n`).

### Readline Timestamp Rendering

The SSH connector does not toggle `UniqueEditLine`. Instead it uses `FuncBeforeSubmit` in the readline fork to append an inline timestamp at submit time, and a painter to colorize just the inline timestamp.

- On Enter, `FuncBeforeSubmit` appends ` (HH:MM:SS)` to the buffer for display and strips it from the submitted line/history.
- If the stamp would split across the line boundary, the connector pads with spaces so `(` starts at the next line; because it is part of the buffer, readline handles the wrapping correctly.
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
