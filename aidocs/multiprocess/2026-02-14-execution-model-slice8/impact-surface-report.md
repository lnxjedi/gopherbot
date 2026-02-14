# Impact Surface Report

## 1) Change Summary

- Slice name: `execution-model-slice8`
- Goal: harden child RPC lifecycle with cancellation/timeouts/error classes and restore admin `ps/kill` parity for interpreter-backed tasks.
- Out of scope:
  - connector routing/identity behavior changes
  - startup mode/config precedence changes
  - migration of compiled-in `taskGo`/`bot/*` handlers (remain in-process)

## 2) Subsystems Affected (with file anchors)

- `bot/pipeline_rpc_lua.go`
- `bot/pipeline_rpc_javascript.go`
- `bot/pipeline_rpc_yaegi.go`
- `bot/calltask.go`
- `bot/pipecontext.go`
- `bot/replyprompt.go`
- `bot/builtins.go`
- `bot/replyprompt_test.go`
- `aidocs/EXECUTION_SECURITY_MODEL.md`
- `aidocs/multiprocess/ARCHITECTURE_DECISIONS.md`

## 3) Current Behavior Anchors

- Interpreter RPC task execution already routes through `runPipelineRPCRequest(...)`.
- Admin `ps`/`kill` inspects/kills `worker.osCmd` process groups.
- Prompt waiters are keyed by protocol/user/channel/thread and removed on timeout/reply/shutdown.

## 4) Proposed Behavior

- What changes:
  - RPC child process is now tracked in `worker.osCmd` during interpreter task execution.
  - RPC request cancel handle is stored in worker state and invoked by admin `kill`.
  - Reply waiters carry task ID; admin `kill` interrupts targeted waiters immediately.
  - RPC lifecycle now uses bounded waits (hello/request/shutdown/exit) with explicit error classes.
- What does not change:
  - compiled-in `taskGo`/`bot/*` remains in-process.
  - connector mapping/routing/authorization semantics remain unchanged.
  - startup/config precedence remains unchanged.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes
- Multi-connector isolation preserved (if applicable)?: yes

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none.
- Config loading/merge/precedence impact:
  - none.
- Execution ordering impact:
  - none; task sequencing remains unchanged.
- Resource lifecycle impact:
  - RPC child lifecycle now explicitly bounded and cancelable.

## 7) Concurrency Risks

- Shared state touched:
  - `worker.osCmd`, `worker.rpcCancel`, `worker.activeTaskTID`.
  - `replies.m` waiter map for targeted interruption.
- Locking/channel/event-order assumptions:
  - worker lock continues to protect pipeline runtime fields.
  - replies lock protects waiter map mutation.
- Race/deadlock/starvation risks:
  - prompt interruption while dispatch/reply timeout paths are active.
- Mitigations:
  - lock-scoped waiter extraction + non-blocking channel sends.
  - idempotent cancel behavior and process-group termination fallback.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - none expected.
- Behavior changes for operators/users:
  - interpreter-backed active pipelines now expose killable PID in `admin ps`.
  - `admin kill` can terminate prompt-blocked interpreter tasks promptly.
- Migration/fallback plan:
  - revert slice 8 commit(s) to restore prior RPC lifecycle behavior.

## 9) Validation Plan

- Focused tests:
  - `go test ./bot`
  - `go test ./bot -run TestInterruptReplyWaitersForTask`
- Broader regression tests:
  - `TEST=GoFull make integration`
  - `TEST=JSFull make integration`
  - `TEST=LuaFull make integration`
- Manual verification steps:
  - start robot with ssh connector, trigger prompt-based interpreter task, verify `admin ps` shows PID and `admin kill` interrupts task.

## 10) Documentation Plan

- `aidocs/EXECUTION_SECURITY_MODEL.md` updates:
  - document RPC child tracking/cancellation and bounded lifecycle waits.
- Other docs:
  - `aidocs/multiprocess/ARCHITECTURE_DECISIONS.md`
