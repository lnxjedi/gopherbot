# AIDEV + MCP (Developer Guide)

This doc explains how to run the AIDEV interface and operate `gopherbot-mcp` in
**one-shot** runs. Treat MCP like `make`: run once, inspect output, decide the
next run. Do not script or keep MCP running.

## Build

```
make
```

This builds both:
- `gopherbot`
- `gopherbot-mcp`

## Run (human operator)

### 1) Start gopherbot in AIDEV mode

From the directory where you want the demo robot to run:

```
/path/to/gopherbot --aidev
```

Gopherbot writes `.mcp-connect` in the working directory with the URL + token:

```
{"url":"http://127.0.0.1:<PORT>","key":"<TOKEN>"}
```

The port and token are **new per run**.

### 2) Run gopherbot-mcp (one-shot)

Use the connect file directly:

```
gopherbot-mcp --connect-file .mcp-connect
```

The MCP process sends:
- `hello` to `/aidev/control`
- `ready` once it connects to `/aidev/stream`

### 3) Confirm readiness

Use `fetch_events` to see the recent event history and confirm the robot is
running.

## Run (AI operator / Codex)

If you want Codex to run MCP commands for you:

1) Start gopherbot with `--aidev` (as above).  
2) Share the `.mcp-connect` contents (or derived command) with Codex.  
3) Codex runs **one-shot** MCP commands and decides the next step after each.

Example prompt to Codex:

```
./gopherbot-mcp --connect-file ../empty-test/.mcp-connect
```

### Prompting tip (channel selection)

Be explicit about the channel. If you want a command to run in `#general`, say
so explicitly (e.g., "send `floyd, tell me a knock-knock joke` in the `general`
channel"). If you omit the channel, the AI may default to a direct message.

## One-shot operator workflow (required)

**Do not** write Python/bash/expect scripts to chain multiple MCP calls. The
correct workflow is manual, sequential one-shot runs with thinking time between
runs.

There are two standard patterns:

**A) History check**
- Re-read `.mcp-connect` to confirm the active session.
- Run MCP once and call `fetch_events`.
- Stop and decide whether to act.

**B) Send + wait**
- Re-read `.mcp-connect` to confirm the active session.
- Run MCP once, call `send_message`, then `wait_for_event` with a short timeout
  (<= 14s).
- Stop and decide the next action.

In short one-shot runs, outbound events can be missed by `wait_for_event`.
Treat the backlog (`fetch_events`) as the consistent source of truth.

Important: `wait_for_event` should **not** be run by itself in a separate
one-shot invocation. If you need to wait, do it in the same MCP run as the
`send_message` that triggered the response. If you reconnect later, use
`fetch_events` to pick up what already happened.

## MCP Tools (minimal)

### send_message
Inject a message as a mapped user.

Arguments:
- `user` (string) — must exist in MCP user map
- `channel` (string) — required unless `direct=true`
- `message` (string)
- `direct` (bool)
- `thread` (string, optional)

### wait_for_event
Wait for the next AIDEV tap event (inbound/outbound).

Arguments:
- `timeout_ms` (number, optional)
- `direction` (string, optional)
- `user` (string, optional)
- `channel` (string, optional)

### fetch_events
Fetch queued AIDEV events since the last fetch (backlog queue).

Arguments:
- `since` (string, optional event ID). If omitted, uses the last event ID seen by MCP.

### control_exit / control_force_exit
Request graceful exit or a forced exit with stack dump.

## User Maps (required for impersonation)

`gopherbot-mcp` has built-in defaults for the terminal connector users:
`alice`, `bob`, `carol`, `david`, `erin`.

To add Slack users, create a YAML file and pass it with `--config`:

```
usermaps:
  slack:
    alice: U0123ABCDE
    bob: U0456FGHIJ
```

Optional environment variables (gopherbot side) that are written into `.mcp-connect`:
- `GOPHER_AIDEV_MCP_CONFIG`
- `GOPHER_AIDEV_MCP_PROTOCOL`

## Notes / Troubleshooting

- `/aidev/inject` returns **409** until the robot is fully initialized (post-connect config loaded).
- The terminal connector yields two related events for an injection: the rewritten
  inbound user message plus the loopback injection message. Use the backlog to
  confirm ordering.
- `fetch_events` returns event IDs formatted as `<NNNNNN>/<HH:MM:SS>`; the queue is
  capped at 1024 items and expires after ~7 minutes.
- AIDEV logs go to the normal robot log; `.mcp-connect` contains the URL/token
  for MCP.
