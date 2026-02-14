# Impact Surface Report

## 1) Change Summary

- Slice name: `execution-model-slice1`
- Goal: introduce an explicit task-execution boundary in the pipeline runner so future slices can move non-`bot/*` task execution to child processes without changing routing/authorization semantics.
- Out of scope:
  - any fork/exec child-runner implementation
  - any IPC protocol design/implementation details
  - changing task authorization/elevation behavior
  - changing admin `ps/kill` behavior in this slice

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/run_pipelines.go`
  - `bot/calltask.go`
  - `bot/task_execution.go` (new)
  - `bot/task_execution_test.go` (new/updated)
  - `aidocs/COMPONENT_MAP.md`
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
- Key functions/types/symbols:
  - `worker.runPipeline(...)` in `bot/run_pipelines.go`
  - `worker.callTask(...)` / `worker.callTaskThread(...)` in `bot/calltask.go`
  - new execution selector boundary on `worker` (slice 1 in-process only)

## 3) Current Behavior Anchors

- Startup/order anchors:
  - `Start(...)` and `run()` startup sequence remains unchanged (`bot/start.go`, `bot/bot_process.go`).
- Routing/message-flow anchors:
  - connector inbound -> `worker.handleMessage()` -> matcher dispatch -> `worker.startPipeline()` -> `worker.runPipeline()`.
- Identity/authorization anchors:
  - authorization/elevation/admin checks happen in `worker.runPipeline()` before task invocation.
- Connector behavior anchors:
  - connector transport behavior and message ordering remain untouched.

## 4) Proposed Behavior

- What changes:
  - add a dedicated execution decision point in the pipeline path (`runPipeline` -> `executeTask`), currently mapped to existing in-process `callTask` behavior.
  - encode the invariant that tasks implemented in `bot/*` (compiled-in Go tasks/plugins/jobs) are always executed in-process.
- What does not change:
  - task results/return codes
  - task ordering and pipeline semantics
  - authorization/elevation checks
  - connector logic, identity resolution, and startup behavior

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes (execution boundary is explicit in pipeline path)
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes (no config surface changes in slice 1)
- Multi-connector isolation preserved (if applicable)?: yes

No invariant redefinition in this slice.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none in slice 1.
- Config loading/merge/precedence impact:
  - none in slice 1.
- Execution ordering impact:
  - none intended; invocation path is refactored through an explicit boundary.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - no new long-lived resources in slice 1.

## 7) Concurrency Risks

- Shared state touched:
  - existing worker/task state only.
- Locking/channel/event-order assumptions:
  - unchanged from current `callTask` model.
- Race/deadlock/starvation risks:
  - low; primarily refactor risk in invocation path.
- Mitigations:
  - keep boundary thin and deterministic
  - run focused bot package tests + integration suite

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - none.
- Behavior changes for operators/users:
  - none expected.
- Migration/fallback plan:
  - n/a for slice 1 (internal refactor groundwork only).

## 9) Validation Plan

- Focused tests:
  - `go test ./bot`
  - add/update focused tests for execution boundary selection and delegation behavior.
- Broader regression tests:
  - `make integration`
- Manual verification steps:
  - n/a for slice 1 (no operator-visible behavior changes expected).

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - none expected for slice 1.
- `aidocs/COMPONENT_MAP.md` updates:
  - add execution-boundary file anchor.
- Connector doc updates:
  - none.
- Other docs:
  - update `aidocs/EXECUTION_SECURITY_MODEL.md` to include the new execution boundary and “bot/* always in-process” invariant.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
