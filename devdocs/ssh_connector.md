# SSH Connector (Design Notes)

This document describes the planned SSH connector that replaces the terminal connector as the default for local development. It focuses on behavior, configuration, and UX.

## Goals

- Default connector for local development (including IDE mode), replacing the terminal connector in default selection paths.
- Maintain test-dev default behavior (terminal connector) for testbot use.
- Provide a lightweight, local SSH chat server with a familiar terminal-like UX.
- Support multiple concurrent SSH clients, each mapped to a single configured user by key.

## High-Level Behavior

- The connector runs an SSH server (Go `x/crypto/ssh`).
- Each connection is authenticated by public key.
- Each user has exactly one authorized key. The key line is stored in the roster as a full `ssh-ed25519 ...` line (comment ignored for matching).
- A bot user is auto-added using the server public key as its UserID.
- One goroutine accepts connections; each client runs in its own goroutine.
- Messages are sent to all connected clients based on each client’s current filter.

## Default Selection (Startup Modes)

When the engine selects a default connector (no `GOPHER_PROTOCOL` provided), SSH should be the default for:

- `demo`
- `ide`
- `production` (if default logic chooses terminal)

Exceptions (remain terminal):

- `test-dev` (testbot)

Any startup modes that currently force `nullconn` remain unchanged.

## Configuration (ProtocolConfig)

### Fields

- `ListenHost` (default: `localhost`)
  - Accepts `localhost`, IPv4 address, IPv6 address, or `all`.
- `ListenPort` (default: `4221`)
  - Overridden by CLI `--ssh-port` and env `GOPHER_SSH_PORT`.
- `HostKey`
  - Optional encrypted host key (private key). If absent, generate ephemeral host key per run.
- `ReplayBufferSize` (default: `42`)
- `MaxMsgBytes` (default: `16384`)
  - Truncated to 4k for buffer
- `DefaultChannel` (default: `general`)
- `BotName` (default: `gopherbot`)
- `Channels` (optional list of valid channel names)
- `Color` (default: `true` in stock config)
- `ColorScheme` (ANSI 256 color map; keys: `prompt`, `timestamp`, `bot`, `user`, `system`, `info`, `warning`, `error`, `private`)

### Defaults

The default SSH connector configuration should exist in `conf/ssh.yaml` (where no port is specified, and the engine defaults to 4221) and `robot.skel/conf/ssh.yaml` (with `ListenPort` using env default `GOPHER_SSH_PORT` or `4221`). The default roster includes alice/bob/carol/david/erin with their ed25519 public keys.

### Color

The stock config enables ANSI 256 colors. User input stays in the terminal default color; prompts, system lines, and bot/user output are colorized based on `ColorScheme`.

Message headers (timestamp/user/channel) are colorized per segment to improve scanability.

## CLI and Environment Overrides

- Add `--ssh-port` to CLI.
- `GOPHER_SSH_PORT` should influence default config.

## .ssh-connect File

When the SSH server successfully binds, write `$CWD/.ssh-connect` with:

```
BOT_SSH_PORT=127.0.0.1:4221
BOT_SERVER_PUBKEY=ssh-ed25519 AAAA... 
```

- `BOT_SSH_PORT` includes host and port.
- `BOT_SERVER_PUBKEY` is the full host public key line.

Also emit an info log that the key was auto-generated (if generated) and log the public key.

## User Experience

### Prompt

Each client sees a prompt:

- `@alice/#general -> `
- `@alice/#general(0005) -> ` when typing in a thread

### Message Output

Messages from other users:

```
(09:15:45)@alice/#general(0005): <message text>
```

Messages from the bot use `=@name`:

```
(09:15:45)=@floyd/#general(0005): <message text>
```

Omit `(0005)` when the message is not threaded.

Hidden messages (only bot/user, not buffered) are prefixed:

```
(private/09:15:45)=@floyd/#general(0005): <message>
```

### Input Echo

- User keystrokes are echoed normally by the PTY.
- On Enter (message send), echo `(timestamp)\n` so the user sees when it was sent.

### Commands

- `|c?` list channels; `|c<name>` switch channel
- `|t?` thread help; `|t` toggle thread; `|t<id>` set thread id
- `|j` join last thread seen
- `|f?` or `|f` list filters; `|fA|fC|fT` set filter

No `|u` command; user identity is fixed by key.

### Filters

Initial filter defaults to Thread (`T`).

- `A` (All): all messages from all channels/threads
- `C` (Channel): all messages in current channel (including threads)
- `T` (Thread): thread-only when in a thread; otherwise channel-level only

### Message Size

User input:

- Accept up to 16k bytes; drop larger with `(ERROR: message too long; > 16k - dropped)`.
- For inputs > 4k, buffer a truncated 4k version and emit `(WARNING: message truncated to 4k in buffer)` to the client.

Bot output:

- Send full output to connected clients.
- Buffer a 4k truncated copy if needed.

### Buffer

- Circular buffer of the last `ReplayBufferSize` messages.
- Only non-hidden messages are buffered.
- On connect, after choosing filter, replay matching buffered messages in arrival order, with timestamps.

## Hidden Messages

- `/botname ...` is treated as hidden (only bot and user can see).
- `/ foo` is treated as a private note-to-self; emit `(INFO: '/' note to self message not sent to other users)`.
- Hidden messages are not added to the replay buffer.

## Paste Handling

- Enable bracketed paste mode on connect; disable on disconnect.
- Paste data is treated as a single message even if it contains newlines.
- For non-bracketed paste, line-based input is used.

## Logging

- No connector-level logging to client output; logs go to stdout/robot.log as usual.
- SSH connector should not call `SetTerminalWriter`.

## Testbot Exception

`make testbot` should still default to the terminal connector (as in `test-dev` mode).

## Helper Script

A helper script `bot-ssh <user>` should exist to simplify local dev, using `KnownHostsCommand` with the `.ssh-connect` info.
