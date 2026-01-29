# AI Operator Model (Robots + Sessions)

This document captures the **conceptual model** that an AI operator (Codex) should keep in mind when working on Gopherbot. It is intentionally stable and is meant to reduce re-orientation costs between sessions.

## Core Concepts

- **Robot**: A long-running Gopherbot process with its own runtime state, configuration, and connector.
- **Session**: A single running robot process. Sessions are **ephemeral** unless the brain is configured to be persistent.
- **Session Handle**: The AIDEV connection details (URL + token) for a running robot session. A new token implies a new session.
- **State**:
  - **Ephemeral state** lives in process memory and is lost when the robot exits.
  - **Persistent state** lives in the configured brain (e.g., Cloudflare KV, DynamoDB) and survives restarts.
  - **Code + config** live in a repo and define the robot’s behavior (plugins, jobs, tasks, config).

## Operational Rules (AI)

- Treat the **AIDEV token** as the **active session identity**.
- Do not mix tokens: a different token means a **different session** (even if run from the same repo).
- When asked to interact with a robot, first locate `.mcp-connect` in that robot’s working directory and use the **token in that file**.
- Always assume the robot process is long-running and stateful; only assume persistence if a persistent brain is configured.
- If a task references a specific robot, identify:
  - Repo path
  - Working directory (where the process was started)
  - Connector/protocol in use
  - Brain type (mem vs persistent)

## Session Readiness Checklist (AI)

- Confirm the `.mcp-connect` token is from the **current** running robot session (new token == new session).
- Start MCP with the explicit URL/key from `.mcp-connect` (do not assume).
- Call `fetch_events` first to drain the backlog and confirm the robot is running; this is the required first step for any new interaction.
- Use `send_and_fetch` for back-and-forth; it waits for an inbound event from a different user (excluding the `aidev` loopback) before returning. If you suspect more messages are coming, call `fetch_events` again (each call drains the queue).
- Proceed with `send_message` only after you see recent events (e.g., welcome/help output).
- If the first reply is a preamble, partial answer, or otherwise not directly actionable, **wait and `fetch_events` again** before responding. Treat “sounds like more is coming” as a cue to pause and drain the queue.
- AIDEV event delivery is **delete-on-read** and assumes a **single consumer**; do not run multiple MCP clients against the same robot session.

## Addressing the Robot (AI)

- Default assumption: commands must be addressed to the robot by **name or alias** (e.g., `floyd, ...` or `;help`).
- A bare `help` is often implemented as an ambient matcher and may be answered by **all robots** in the channel; use it to discover names/aliases when unsure.
- Bare commands like `help list` only work when there are **ambient message matchers**, which are uncommon; do not assume they exist.
- If a help response explicitly shows the correct addressing format, follow it exactly for subsequent commands.

## One-shot MCP Runs (Recommended)

Operate `gopherbot-mcp` as short, single-purpose invocations rather than a
long-lived interactive client. This keeps state handling simple and aligns with
the event queue behavior.

Two standard patterns:

**History check**
- Re-read `.mcp-connect` to confirm the active session.
- Run MCP once to `fetch_events`.
- Stop and decide the next action.

**Send + fetch**
- Re-read `.mcp-connect` to confirm the active session.
- Run MCP once to `send_and_fetch` with a short timeout (<= 14s).
- Stop and decide the next action based on what you received.

Do not run `send_message` then wait separately. If you need a reply, use
`send_and_fetch`. Later polling should use `fetch_events`.

Avoid writing expect-style scripts; treat each MCP run like a single "make"
invocation: run it, inspect output, decide the next run.

See `aidocs/mcp-integration.md` for detailed MCP design notes and operator
sequencing guidance.

Explicit rule: **Do not** create Python/bash scripts to automate multiple MCP
interactions. The operator workflow is deliberately manual and iterative, one
MCP run at a time.

For the practical MCP runbook (commands + prompting tips), see
`devdocs/aidev-mcp.md`.

## Example Prompt Clarity (Human → AI)

Good:
- “We’re working on Bishop in `../bishop-gopherbot`, started from that directory with `--aidev`; use the `.mcp-connect` in that directory.”
- “This robot uses Cloudflare KV, so state should persist across restarts.”

Bad (ambiguous):
- “Let’s talk to the robot” (no directory, no token, no session identity)

## Summary

A robot is a **running process**. A session is defined by the **AIDEV token**. Persistent brain configuration determines what survives across restarts. The AI should always anchor itself to a **specific working directory + `.mcp-connect`** before using MCP tools.
