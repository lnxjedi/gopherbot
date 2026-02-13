# Impact Surface Report

## 1) Change Summary

- Slice name: `log-protocol-context-slice3`
- Goal: Improve operator diagnostics by adding explicit protocol context to prompt timeout/retry and similar channel/thread log messages.
- Out of scope:
  - behavior changes to prompt matching, routing, or authorization
  - connector API changes

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/replyprompt.go`
  - `bot/dispatch.go`
  - `bot/subscribe_thread.go`
  - `aidocs/multi-protocol/2026-02-13-log-protocol-context-slice3/*`
- Key functions/types/symbols:
  - `Robot.promptInternal(...)`
  - dispatch reply waiter/catchall/subscription logging paths
  - `Robot.Subscribe()` / `Robot.Unsubscribe()`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - none.
- Routing/message-flow anchors:
  - no control-flow changes.
- Identity/authorization anchors:
  - none.
- Connector behavior anchors:
  - none.

## 4) Proposed Behavior

- What changes:
  - warning/debug logs in prompt timeout and related channel/thread logs include protocol.
- What does not change:
  - prompt timeout durations and matcher semantics.
  - dispatch routing behavior and subscription behavior.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes
- Multi-connector isolation preserved (if applicable)?: yes

No invariant is redefined.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none.
- Config loading/merge/precedence impact:
  - none.
- Execution ordering impact:
  - none.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - none.

## 7) Concurrency Risks

- Shared state touched:
  - none beyond existing read paths.
- Locking/channel/event-order assumptions:
  - unchanged.
- Race/deadlock/starvation risks:
  - none added.
- Mitigations:
  - logging-only edits.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - none.
- Behavior changes for operators/users:
  - clearer log lines with protocol context.
- Migration/fallback plan:
  - n/a.

## 9) Validation Plan

- Focused tests:
  - compile/test regression only (no logic change).
- Broader regression tests:
  - `go test ./bot`
  - `go test ./...`
- Manual verification steps:
  - trigger prompt timeout and verify protocol appears in warning line.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - n/a.
- `aidocs/COMPONENT_MAP.md` updates:
  - n/a.
- Connector doc updates:
  - n/a.
- Other docs:
  - slice checklist + compatibility note.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
