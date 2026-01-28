# AIDEV + MCP (Developer Guide)

This doc explains how to run the AIDEV interface and `gopherbot-mcp`, and how to tell the AI (Codex) what command to run.

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

Gopherbot will log a line to **stderr** that looks like:

```
AIDEV: start MCP with: /path/to/gopherbot-mcp --aidev-url http://127.0.0.1:<PORT> --aidev-key <TOKEN>
```

This line includes the **random port** and **random token** for the current run.

### 2) Start gopherbot-mcp

Run exactly the command printed by gopherbot:

```
/path/to/gopherbot-mcp --aidev-url http://127.0.0.1:<PORT> --aidev-key <TOKEN>
```

The MCP process will send:
- `hello` to `/aidev/control`
- `ready` once it connects to `/aidev/stream`

### 3) Tell the robot to start

The robot waits for a `start` control command before starting the connector. You can trigger this via MCP:

Tool: `start_robot`

Once started, you should see the welcome message in the terminal connector output.

## Run (AI operator / Codex)

If you want Codex to run MCP commands for you:

1) Start gopherbot with `--aidev` (as above).  
2) Copy the **exact MCP start command** from stderr.  
3) Paste that command into the chat so Codex can run it.

Example prompt to Codex:

```
./gopherbot-mcp --aidev-url http://127.0.0.1:35223 --aidev-key 9bb0a17f...
```

Codex will then:
- run MCP
- call `start_robot`
- call `send_message` (for example, `;exit` as `alice`)

## MCP Tools (minimal)

### start_robot
Releases the startup gate so the connector + post-connect config run.

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

### control_exit / control_force_exit
Request graceful exit or a forced exit with stack dump.

## User Maps (required for impersonation)

`gopherbot-mcp` has built‑in defaults for the terminal connector users:
`alice`, `bob`, `carol`, `david`, `erin`.

To add Slack users, create a YAML file and pass it with `--config`:

```
usermaps:
  slack:
    alice: U0123ABCDE
    bob: U0456FGHIJ
```

Optional environment variables (gopherbot side) that will appear in the MCP start command:
- `GOPHER_AIDEV_MCP_CONFIG`
- `GOPHER_AIDEV_MCP_PROTOCOL`

## Notes

- `/aidev/inject` returns **409** until the robot is fully initialized (post-connect config loaded).
- For the terminal connector, injected messages are also looped directly into `IncomingMessage` so they behave like real user input.
