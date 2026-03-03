# Impact Surface Report

## 1) Change Summary

- Slice name: `execution-model-slice4`
- Goal: introduce a minimal, versioned stdio RPC protocol scaffold and internal child command path for future interpreter migration, without changing runtime task execution behavior.
- Out of scope:
  - moving interpreter-backed external tasks (`.go`/`.lua`/`.js`) to child mode
  - changing `executeTask` runner selection
  - changing connector/identity/authorization behavior

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/start.go`
  - `bot/task_execution_child.go`
  - `bot/pipeline_rpc.go` (new)
  - `bot/task_execution_child_test.go`
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/COMPONENT_MAP.md`
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/multiprocess/2026-02-14-execution-model-slice4/*`
- Key functions/types/symbols:
  - startup internal child command dispatch in `Start(...)`
  - RPC protocol message structs / codec helpers
  - `runPipelineChildRPC` stdio loop

## 3) Current Behavior Anchors

- Startup/order anchors:
  - `pipeline-child-exec` fast path in `Start(...)`
- Routing/message-flow anchors:
  - `runPipeline` -> `executeTask` remains unchanged
- Identity/authorization anchors:
  - all auth/policy and identity mapping remain engine-side and unchanged
- Connector behavior anchors:
  - connector runtime remains parent-only

## 4) Proposed Behavior

- What changes:
  - add internal command `pipeline-child-rpc` parsed early in startup.
  - add minimal stdio protocol scaffolding: versioned message envelope, `hello` handshake, and `shutdown` method.
- What does not change:
  - no execution path uses the new RPC command yet.
  - no behavior change for operators/users.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes
- Multi-connector isolation preserved (if applicable)?: yes

No invariant redefinition is proposed in this slice.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - add one internal command branch after `flag.Parse()`; normal startup order unchanged.
- Config loading/merge/precedence impact:
  - none.
- Execution ordering impact:
  - none (scaffold not wired into task execution).
- Resource lifecycle impact:
  - child RPC command is short-lived and exits on EOF/shutdown.

## 7) Concurrency Risks

- Shared state touched:
  - none (local stdio codec state only).
- Locking/channel/event-order assumptions:
  - none across shared engine state.
- Race/deadlock/starvation risks:
  - low; simple sync decode/encode loop.
- Mitigations:
  - bounded protocol handling with explicit error responses/exit codes.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - none.
- Behavior changes for operators/users:
  - none.
- Migration/fallback plan:
  - n/a (internal scaffolding only).

## 9) Validation Plan

- Focused tests:
  - unit tests for protocol encode/decode and handshake loop.
- Broader regression tests:
  - `go test ./bot`
  - `make integration`
- Manual verification steps:
  - invoke `./gopherbot pipeline-child-rpc` with a hello message and verify reply.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - document `pipeline-child-rpc` internal fast path.
- `aidocs/COMPONENT_MAP.md` updates:
  - include new RPC scaffold file.
- Connector doc updates:
  - none.
- Other docs:
  - update execution model doc to state RPC scaffold exists but is not yet active in runtime selection.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
