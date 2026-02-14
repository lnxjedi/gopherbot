# Impact Surface Report

## 1) Change Summary

- Slice name: `execution-model-slice3`
- Goal: route external executable plugin default-config retrieval (`configure`) through the same child process runner (`gopherbot pipeline-child-exec`) introduced in slice 2, so external executable execution is consistently out-of-process in both runtime pipeline execution and config-load configure paths.
- Out of scope:
  - moving interpreter-backed external tasks (`.go`/`.lua`/`.js`) to child mode
  - changing `taskGo` execution model (must remain in-process)
  - connector/runtime/identity behavior changes

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/calltask.go`
  - `bot/task_execution_child.go`
  - `bot/task_execution_child_test.go`
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/multiprocess/2026-02-14-execution-model-slice3/*`
- Key functions/types/symbols:
  - `getDefCfgThread` (`bot/calltask.go`)
  - child request encode/decode + command construction helpers (`bot/task_execution_child.go`)
  - `pipelineChildExecRequest`, `pipelineChildExecCommand`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - `Start(...)` fast-path for `pipeline-child-exec` in `bot/start.go`
- Routing/message-flow anchors:
  - runtime task execution is selected via `executeTask` (`bot/task_execution.go`)
  - slice 2 already runs external executable runtime tasks/jobs/plugins in child process
- Identity/authorization anchors:
  - no changes to message identity mapping, authorization, or connector routing
- Connector behavior anchors:
  - connectors remain parent-process responsibilities

## 4) Proposed Behavior

- What changes:
  - external executable plugin `configure` path (default config retrieval during load) uses child runner instead of direct parent `exec.Command(taskPath, "configure")`.
  - add/reuse helper to build child-runner `exec.Cmd` from `pipelineChildExecRequest`.
- What does not change:
  - `taskGo` remains in-process.
  - interpreter-backed external tasks remain in-process.
  - connector runtime, startup order, and identity policy remain unchanged.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes (unchanged)
- Connector ordering guarantees preserved?: yes (unchanged)
- Config precedence still explicit?: yes
- Multi-connector isolation preserved (if applicable)?: yes (unchanged)

No invariant redefinition is proposed in this slice.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none; this touches config-load subprocess behavior but not startup phase order.
- Config loading/merge/precedence impact:
  - no merge/precedence change; only execution mechanism for external executable `configure`.
- Execution ordering impact:
  - same synchronous ordering from caller perspective; still blocks until configure completes.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - one short-lived child process per external executable configure call.

## 7) Concurrency Risks

- Shared state touched:
  - none beyond existing logging and task config loading paths.
- Locking/channel/event-order assumptions:
  - preserve existing behavior in `getDefCfgThread` goroutine.
- Race/deadlock/starvation risks:
  - low; child call is synchronous and bounded by existing command execution lifecycle.
- Mitigations:
  - reuse existing child request validation + exit handling.
  - preserve existing error shaping for configure failures.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - no configuration changes required.
- Behavior changes for operators/users:
  - none intended functionally; execution mechanism is internal.
- Migration/fallback plan:
  - no migration required.

## 9) Validation Plan

- Focused tests:
  - `go test ./bot` (child request/runner tests)
- Broader regression tests:
  - `make integration`
- Manual verification steps:
  - run a robot with an external executable plugin and confirm `configure` still loads defaults correctly.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - not expected (startup flow unchanged).
- `aidocs/COMPONENT_MAP.md` updates:
  - not expected unless helper boundaries materially shift.
- Connector doc updates:
  - none.
- Other docs:
  - update `aidocs/EXECUTION_SECURITY_MODEL.md` to note child-runner coverage now includes external executable `configure` path.
  - add slice-3 checklist and compatibility note artifacts.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
