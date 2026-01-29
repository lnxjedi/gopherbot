# AIDEV + MCP (Developer Guide)

This doc explains how to run the AIDEV interface and operate `gopherbot-mcp` in
**one-shot** runs. Treat MCP like `make`: run once, inspect output, decide the
next run. Do not script or keep MCP running.

## Fresh Session Checklist

1) Re-read `.mcp-connect` to confirm the active session token.
2) Run a one-shot `fetch_events` to establish context and readiness (this drains the queue).
3) Proceed with `send_and_fetch` for back-and-forth.

## Quoting-safe one-shot pattern

When invoking `gopherbot-mcp` from a shell, avoid quote-escaping failures by
feeding JSON via a here-doc:

```bash
cat <<'JSON' | ./gopherbot-mcp --connect-file ../empty-test/.mcp-connect
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"send_and_fetch","arguments":{"user":"david","channel":"general","message":"floyd, help","direct":false,"timeout_ms":12000}}}
JSON
```

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
- `ready` to `/aidev/control`

### 3) Confirm readiness

Use `fetch_events` to see the recent event history and confirm the robot is
running. This endpoint uses delete-on-read semantics, so each fetch drains
the queue.

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
- Run MCP once and call `fetch_events` (optionally `wait_ms` to wait briefly).
- Stop and decide whether to act.

**B) Send + fetch (preferred)**
- Re-read `.mcp-connect` to confirm the active session.
- Always start a new interaction by running the History check first.
- Run MCP once and call `send_and_fetch` with a short timeout (<= 14s).
- Stop and decide the next action. If the first reply is a preamble, partial
  answer, or otherwise not directly actionable, **wait and `fetch_events` again**
  before responding. If you suspect more messages are coming, use `fetch_events`
  with `wait_ms` after. If `send_and_fetch` times out, **do not resend
  immediately**—run `fetch_events` first to avoid duplicate replies.

## MCP Tools (minimal)

### send_message
Inject a message as a mapped user (use sparingly; prefer `send_and_fetch`).

Arguments:
- `user` (string) — must exist in MCP user map
- `channel` (string) — required unless `direct=true`
- `message` (string)
- `direct` (bool)
- `thread` (string, optional)

### send_and_fetch
Send a message and return events fetched during the call (delete-on-read),
waiting until at least one inbound event from a different user than the sender
(excluding the `aidev` loopback user), or timeout.

Note: `send_and_fetch` may return a preamble before the bot’s next prompt. If
the reply sounds incomplete, run `fetch_events` with `wait_ms` before replying.

Arguments:
- `user` (string)
- `channel` (string, optional)
- `thread` (string, optional)
- `message` (string)
- `direct` (bool, optional)
- `timeout_ms` (number, optional, default 14000)

Example (one-shot JSON-RPC):

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"send_and_fetch","arguments":{"user":"david","channel":"general","message":"floyd, tell me a knock-knock joke","direct":false,"timeout_ms":12000}}}
```

### fetch_events
Fetch all queued AIDEV events (delete-on-read), optionally waiting for at least
one event.

Arguments:
- `wait_ms` (number, optional) to wait for at least one event.

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
  capped at 1024 items and expires after ~7 minutes. Each fetch drains the queue.
- AIDEV event delivery is delete-on-read; run only one MCP client per robot session.
- AIDEV logs go to the normal robot log; `.mcp-connect` contains the URL/token
  for MCP.
