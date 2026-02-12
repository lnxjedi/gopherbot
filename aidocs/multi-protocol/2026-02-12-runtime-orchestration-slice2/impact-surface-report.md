# Impact Surface Report

## 1) Change Summary

- Slice name: `runtime-orchestration-slice2`
- Goal: implement simultaneous multi-protocol runtime orchestration with primary + secondary connector lifecycle control.
- Out of scope:
  - cross-protocol message sending API additions for plugins
  - protocol-agnostic shared profile store design
  - thread mapping across protocols

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/start.go`
  - `bot/bot_process.go`
  - `bot/conf.go`
  - `bot/builtins.go`
  - `bot/send_message.go`
  - `bot/robot_connector_methods.go`
  - `conf/plugins/builtin-admin.yaml`
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/COMPONENT_MAP.md`
  - `aidocs/TESTING_CURRENT.md` (if harness notes are affected)
- Key functions/types/symbols:
  - connector init path in `Start(...)` (`bot/start.go`)
  - connector run loop in `run()` (`bot/bot_process.go`)
  - config reload in `loadConfig(false)` (`bot/conf.go`)
  - admin command handler in `admin(...)` (`bot/builtins.go`)
  - send path in `bot/send_message.go`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - one connector selected by `currentCfg.protocol` in `bot/start.go`.
  - one connector run loop in `bot/bot_process.go`.
- Routing/message-flow anchors:
  - all outgoing connector calls go through global `interfaces` connector methods.
- Identity/authorization anchors:
  - listed user + ignore-unlisted behavior in `handler.IncomingMessage` (`bot/handler.go`).
  - admin checks remain username-based (`worker.checkAdmin`, `bot/robot.go`).
- Connector behavior anchors:
  - connector interface currently modeled as single active connector implementation.

## 4) Proposed Behavior

- What changes:
  - add a runtime connector manager wrapper that:
    - keeps primary connector as required
    - attempts startup of all configured `SecondaryProtocols`
    - logs secondary start failures without failing robot startup
    - routes outgoing message sends to source protocol when `msgObject.Protocol` is set; otherwise primary
    - supports reload reconciliation:
      - stop connectors removed from `SecondaryProtocols`
      - attempt start for newly configured secondaries
      - if `PrimaryProtocol` changes on reload, log error and ignore change
    - supports admin lifecycle commands on primary protocol:
      - `protocol-list` / `protocol list`
      - `protocol-start <name>` / `protocol start <name>`
      - `protocol-stop <name>` / `protocol stop <name>`
      - `protocol-restart <name>` / `protocol restart <name>`
  - retry policy: no autonomous timer retry; retries occur only on reload or admin command.
- What does not change:
  - primary failure at startup remains fatal.
  - scheduled jobs continue to use primary protocol by default.
  - thread semantics remain protocol-local.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes (single runtime manager path)
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes per connector (manager does not reorder within connector streams)
- Config precedence still explicit?: yes
- Multi-connector isolation preserved (if applicable)?: yes (secondary exit does not terminate primary runtime)

No invariant redefinition required.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - primary connector still initialized during startup; secondary startup attempts happen under manager runtime path.
- Config loading/merge/precedence impact:
  - loader now applies secondary reconciliation on post-connect reload.
- Execution ordering impact:
  - one goroutine per connector run loop, managed by runtime manager.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - shutdown path must close all connector stop channels; primary loop completion still controls process lifecycle.

## 7) Concurrency Risks

- Shared state touched:
  - runtime manager connector state map
  - desired secondary protocol set
  - cached user map for connector start-after-load
- Locking/channel/event-order assumptions:
  - explicit mutex around state transitions (start/stop/reconcile/list).
  - avoid double-close on stop channels.
- Race/deadlock/starvation risks:
  - concurrent reload and admin protocol commands could race start/stop.
- Mitigations:
  - serialized state mutation under manager lock.
  - idempotent start/stop checks and status transitions.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - robots without `SecondaryProtocols` keep single-connector behavior.
  - legacy `Protocol` support remains.
- Behavior changes for operators/users:
  - new admin commands become available for protocol lifecycle.
  - secondary connector failures become non-fatal and visible via logs/status.
- Migration/fallback plan:
  - remove `SecondaryProtocols` to return to single-connector runtime while preserving primary behavior.

## 9) Validation Plan

- Focused tests:
  - runtime manager start/reconcile/list/stop behavior unit tests
  - reload behavior test: primary change ignored with log
  - send routing test by source protocol
- Broader regression tests:
  - `go test ./...`
  - integration harness sanity (`test` connector flow unchanged)
- Manual verification steps:
  - primary ssh + secondary slack configured: startup attempts both
  - remove secondary and run reload: secondary stops
  - failed secondary start + reload: retries once per reload
  - admin commands from primary channel control secondaries

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - document runtime manager orchestration and reload behavior.
- `aidocs/COMPONENT_MAP.md` updates:
  - add runtime manager under `bot/`.
- Connector doc updates:
  - `aidocs/SSH_CONNECTOR.md`/Slack docs if lifecycle assumptions changed.
- Other docs:
  - keep `aidocs/multi-protocol/...` slice artifacts current.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
