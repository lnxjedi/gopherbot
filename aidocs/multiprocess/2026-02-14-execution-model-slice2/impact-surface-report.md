# Impact Surface Report

## 1) Change Summary

- Slice name: `execution-model-slice2`
- Goal: move external executable task/plugin/job execution (non-`.go`/`.lua`/`.js`) to fork+exec child gopherbot processes via the `executeTask` boundary.
- Out of scope:
  - moving interpreter-backed tasks (`yaegi`, Lua, JS) to child mode
  - changing `taskGo` execution model (remains in-process)
  - long-lived worker pools
  - parent/child asynchronous messaging protocol beyond single-request execution

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/task_execution.go`
  - `bot/calltask.go`
  - `bot/start.go`
  - `bot/task_execution_child.go` (new)
  - `bot/task_execution_test.go`
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/COMPONENT_MAP.md`
  - `aidocs/STARTUP_FLOW.md`
- Key functions/types/symbols:
  - `worker.selectTaskExecutionRunner(...)` and `worker.executeTask(...)`
  - `worker.callTask(...)` / `worker.callTaskThread(...)`
  - internal child command entrypoint in startup path (`Start(...)`)

## 3) Current Behavior Anchors

- Startup/order anchors:
  - `Start(...)` parses flags and runs full startup unless CLI command path is chosen (`bot/start.go`).
- Routing/message-flow anchors:
  - connector inbound -> dispatch -> `startPipeline` -> `runPipeline` -> `executeTask`.
- Identity/authorization anchors:
  - authorization/elevation checks are executed before task invocation in `runPipeline`.
- Connector behavior anchors:
  - connector runtime is parent-process only.

## 4) Proposed Behavior

- What changes:
  - add internal child-runner command (`pipeline-child-exec`) handled early in startup.
  - route external executable tasks through a process runner that spawns `gopherbot pipeline-child-exec`.
  - child runner executes exactly one external command, streams stdout/stderr, returns command exit status, then exits.
  - parent remains owner of pipeline state/log/history/auth/brain/connector behavior.
- What does not change:
  - `taskGo` remains in-process always.
  - interpreter-backed external tasks (`.go`/`.lua`/`.js`) remain in-process in this slice.
  - authorization/elevation policy location remains in engine pipeline flow.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes (single explicit early branch for internal child command)
- Explicit control flow preserved?: yes (`executeTask` chooses runner by task class)
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes (no config changes)
- Multi-connector isolation preserved (if applicable)?: yes

No invariant redefinition.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - add an internal startup fast path for child-runner command that bypasses normal robot initialization.
- Config loading/merge/precedence impact:
  - none.
- Execution ordering impact:
  - none; pipeline task order unchanged.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - one short-lived child process per external executable invocation.

## 7) Concurrency Risks

- Shared state touched:
  - `worker.osCmd` tracking for `ps/kill` during external execution.
- Locking/channel/event-order assumptions:
  - unchanged around pipeline/task state mutation.
- Race/deadlock/starvation risks:
  - medium: pipe copy/wait ordering in parent and child could deadlock if implemented incorrectly.
- Mitigations:
  - concurrent stdout/stderr draining with deterministic wait logic
  - preserve existing process-group kill behavior in parent path
  - focused tests for runner selection and child command parsing

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - no config migration required.
- Behavior changes for operators/users:
  - external executable tasks now run under child gopherbot process; `ps` shows child pid.
  - no command/API surface changes intended.
- Migration/fallback plan:
  - keep in-process fallback path for non-external-executable task types.

## 9) Validation Plan

- Focused tests:
  - `go test ./bot`
  - runner selection tests: executables -> process runner, interpreters -> in-process
  - child request decode/validation unit tests
- Broader regression tests:
  - `make integration`
- Manual verification steps:
  - optional: run a slow external task and verify `ps`/`kill` still work on active pid.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - note internal `pipeline-child-exec` fast path and that it bypasses normal startup.
- `aidocs/COMPONENT_MAP.md` updates:
  - add child execution file anchor.
- Connector doc updates:
  - none.
- Other docs:
  - update `aidocs/EXECUTION_SECURITY_MODEL.md` with slice-2 executable child model.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
