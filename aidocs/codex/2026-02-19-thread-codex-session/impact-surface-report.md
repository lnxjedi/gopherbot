# Impact Surface Report

## 1) Change Summary

- Slice name: `codex-thread-session`
- Goal: add a compiled-in built-in plugin that supports:
  - `/(bot) link-codex` per-user authentication bootstrap without inbound callback requirements
  - `;start-codex <directory>` to start a per-thread Codex app-server session bound to that working directory
  - thread-scoped message relay from Gopherbot <-> Codex until `end-session`
- Out of scope:
  - connector-specific UX customization beyond existing canonical send APIs
  - startup-time Codex process management
  - modifying extension API signatures

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/builtins.go`
  - `bot/bot_process.go`
  - `bot/` (new codex built-in/session files)
  - `conf/plugins/` (new built-in plugin config)
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/COMPONENT_MAP.md`
- Key functions/types/symbols:
  - plugin registration in `bot/builtins.go`
  - lifecycle stop path in `bot/bot_process.go` func `stop`
  - routing and thread subscription behavior from `bot/dispatch.go` and `bot/subscribe_thread.go`
  - connector send routing from `bot/connector_runtime.go`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - `bot/start.go` func `Start`
  - `bot/bot_process.go` funcs `initBot`, `run`, `stop`
- Routing/message-flow anchors:
  - `bot/handler.go` method `IncomingMessage`
  - `bot/dispatch.go` method `handleMessage`
  - `bot/subscribe_thread.go` funcs `Subscribe`, `Unsubscribe`, `lookupSubscriptionLocked`
- Identity/authorization anchors:
  - username resolution in `bot/handler.go` (`resolveIncomingUser`)
  - pipeline auth/admin checks in `bot/run_pipelines.go`
- Connector behavior anchors:
  - runtime routing in `bot/connector_runtime.go`
  - send methods in `robot/connector_defs.go`

## 4) Proposed Behavior

- What changes:
  - New built-in plugin command family for Codex link/start/stop/status plus thread subscription command handling.
  - New background session manager for long-lived per-thread Codex app-server process + JSON-RPC bridge.
  - Per-user Codex auth cache persisted via robot brain datum (username-keyed).
  - Stop path additionally terminates active Codex session processes before connector runtime shutdown completes.
- What does not change:
  - connector interfaces and inbound normalization contract
  - engine routing order and matcher precedence
  - startup mode detection and config precedence rules
  - extension API method signatures

## 5) Invariant Impact Check

- Startup determinism preserved?: yes (no startup-time Codex launch; deterministic stop cleanup only)
- Explicit control flow preserved?: yes (single manager with explicit lifecycle methods)
- Shared auth/policy remains in engine flows?: yes (command entry still via plugin pipeline checks)
- Permission checks remain username-based?: yes (session ownership + auth storage keyed by canonical username)
- Connector ordering guarantees preserved?: yes (session output uses existing per-connector send APIs)
- Config precedence still explicit?: yes (new plugin config follows normal config merge/load)
- Multi-connector isolation preserved (if applicable)?: yes (session keys include protocol and send routing uses protocol context)

If any invariant is redefined, explain why and list required doc updates.
- No invariant redefinition planned.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none for start sequence; shutdown sequence gains Codex session stop step.
- Config loading/merge/precedence impact:
  - additive plugin config only.
- Execution ordering impact:
  - plugin command handlers enqueue work; session worker serializes turn execution per session.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - new goroutines for RPC event loop + per-session worker.
  - explicit process teardown on `end-session` and global robot stop.

## 7) Concurrency Risks

- Shared state touched:
  - global session manager maps (session key -> session state)
  - per-session request/response and event channels
- Locking/channel/event-order assumptions:
  - one active turn per session worker queue
  - single reader loop demultiplexes JSON-RPC responses and notifications
- Race/deadlock/starvation risks:
  - stale session entries after process crash
  - blocked queues under high message volume
  - potential concurrent stop + enqueue races
- Mitigations:
  - bounded channels with backpressure/error responses
  - manager-level locking + idempotent stop
  - periodic/exit cleanup path and explicit done-channel joins

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - additive only; no existing command semantics changed.
- Behavior changes for operators/users:
  - new built-in commands become available.
  - optional new child process footprint when commands are used.
- Migration/fallback plan:
  - if Codex binary/auth unavailable, commands fail with explicit guidance and do not affect existing behavior.

## 9) Validation Plan

- Focused tests:
  - unit tests for session keying and path validation
  - unit tests for auth datum persistence helpers
  - unit tests for manager lifecycle idempotency/cleanup
- Broader regression tests:
  - `go test ./bot -run Codex`
  - `go test ./...` (best effort)
- Manual verification steps:
  - invoke `/(bot) link-codex`
  - start session with `;start-codex <dir>`
  - send threaded message and verify Codex response in same thread
  - `end-session` and confirm process teardown

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - add shutdown cleanup mention for Codex sessions.
- `aidocs/COMPONENT_MAP.md` updates:
  - add new codex built-in/session files under `bot/`.
- Connector doc updates:
  - none expected (connector contracts unchanged).
- Other docs:
  - optional follow-up operator usage note for Codex built-in commands.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
