# SSH Connector (User Guide)

This document describes how to use the built‑in SSH connector for local development.

## Quick Start

1. Start the robot from its repo directory:

```
./gopherbot
```

2. Connect with the helper script (default users are `alice`, `bob`, `carol`, `david`, `erin`):

```
./bot-ssh alice
```

On first connect, you’ll be prompted to choose an initial filter.

## What You’ll See

### Prompt

```
@alice/#general -> 
```

If you’re typing in a thread:

```
@alice/#general(0005) -> 
```

### Message Output

User message:

```
(09:15:45)@alice/#general(0005): hello
```

Bot message:

```
(09:15:45)=@floyd/#general(0005): @alice PONG
```

Hidden (private) reply:

```
(private/09:15:45)=@floyd/#general(0005): ...
```

### Timestamps on Send

When you press Enter, the connector echoes a timestamp so you can see when the message was sent:

```
@alice/#general -> ;ping (09:15:45)
```

## Filters (A / C / T)

When you connect, you’re asked to pick an initial filter. Press `A`, `C`, or `T` (or just Enter for the default).

- **A (All)**: all messages in all channels and threads
- **C (Channel)**: all messages in the current channel (including threads)
- **T (Thread)**:
  - If you’re in a thread: only that thread
  - If you’re not in a thread: channel‑level messages **plus** the first message of each thread

For thread announcements, the first thread message is shown with `(+0005)` so you know a thread was created.

## Connector Commands

Type these commands at the prompt:

- `|c?` list channels
- `|c<name>` change channel (example: `|cgeneral`)
- `|t?` thread help
- `|t` toggle typing in a thread (creates a new thread id)
- `|t<id>` type in a specific thread id
- `|j` join the last thread you saw
- `|f?` list filters
- `|fA` / `|fC` / `|fT` set filter

## Hidden / Private Messages

- `/botname ...` sends a hidden message only visible to you and the bot.
- `/ ...` (slash followed by a space) is a “note to self” and is not sent to others.

Hidden replies are marked with `(private/...)` and are **not** added to the replay buffer.

## Replay Buffer

On connect, the connector replays recent messages that match your filter.

- Default buffer size: 42 messages
- Messages are truncated to 4k in the buffer (live output is full‑length)

## Message Length Limits

- User input max: 16k bytes
  - >16k is dropped with an error
  - >4k is kept in full for live viewers but only 4k is saved in the replay buffer

## Paste Handling

Bracketed paste is enabled, so multi‑line paste works as a single message (where supported by the client).

## Host Key and Known Hosts

When the connector starts, it writes `.ssh-connect` in the current directory with:

```
BOT_SSH_PORT=127.0.0.1:4221
BOT_SERVER_PUBKEY=ssh-ed25519 AAAA...
```

`bot-ssh` uses this to validate the server key automatically.

## Configuration (Quick Reference)

In `conf/ssh.yaml`:

- `ListenHost` (default `localhost`)
- `ListenPort` (default `4221`)
- `DefaultChannel` (default `general`)
- `ReplayBufferSize` (default `42`)
- `MaxMsgBytes` (default `16384`)
- `UserHistoryLines` (default `14`)
- `Color` (default `true` in stock config)
- `ColorScheme` (ANSI 256 colors by key)

## Troubleshooting

- If `./bot-ssh alice` fails, check `.ssh-connect` exists and the robot is running.
- If you can’t connect, verify the user’s public key is in `conf/ssh.yaml` under `UserRoster`.

