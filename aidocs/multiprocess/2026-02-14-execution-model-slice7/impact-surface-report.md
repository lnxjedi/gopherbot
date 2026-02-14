# Impact Surface Report

## 1) Change Summary

- Slice name: `execution-model-slice7`
- Goal: move external Go interpreter-backed plugin/job/task execution and Go plugin default-config retrieval onto the generic `pipeline-child-rpc` contract.
- Out of scope:
  - changes to connector behavior/routing/identity mapping
  - startup mode/config precedence changes
  - migration of compiled-in `taskGo`/`bot/*` handlers (remain in-process)

## 2) Subsystems Affected (with file anchors)

- `bot/calltask.go`
- `bot/pipeline_rpc.go`
- `bot/pipeline_rpc_yaegi.go` (new)
- `bot/pipeline_rpc_lua.go` (expanded `robot_call` forwarding surface used by Go client)
- `aidocs/STARTUP_FLOW.md`
- `aidocs/COMPONENT_MAP.md`
- `aidocs/EXECUTION_SECURITY_MODEL.md`
- `aidocs/multiprocess/ARCHITECTURE_DECISIONS.md`

## 3) Current Behavior Anchors

- Startup child-command fast paths remain in `Start(...)` (`pipeline-child-exec`, `pipeline-child-rpc`).
- Parent engine remains policy/routing/identity authority; child requests Robot operations via `robot_call`.
- Lua and JavaScript were already on generic RPC from prior slices.

## 4) Proposed Behavior

- What changes:
  - external Go plugin execution uses `go_plugin_run` over child RPC.
  - external Go job execution uses `go_job_run` over child RPC.
  - external Go task execution uses `go_task_run` over child RPC.
  - external Go plugin default-config retrieval uses `go_get_config` over child RPC.
- What does not change:
  - compiled-in `taskGo`/`bot/*` remains in-process.
  - connector, authorization, identity, and routing semantics remain unchanged.
  - startup/config precedence behavior remains unchanged.

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
  - none; internal child RPC command path already exists.
- Config loading/merge/precedence impact:
  - none; execution transport changed without config-shape change.
- Execution ordering impact:
  - request/response remains synchronous per task invocation.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - one-shot child RPC lifecycle remains start/hello/request/shutdown/wait.

## 7) Concurrency Risks

- Shared state touched:
  - parent-side Robot state accessed through existing `robot_call` path.
- Locking/channel/event-order assumptions:
  - response correlation by request ID remains required.
- Race/deadlock/starvation risks:
  - potential deadlock if `robot_call` handling blocks indefinitely.
- Mitigations:
  - strict request/response framing and existing shutdown path per child.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - none expected; Go extension behavior should be equivalent.
- Behavior changes for operators/users:
  - no command/config changes.
- Migration/fallback plan:
  - revert slice 7 commit(s) to return Go external execution to prior in-process path.

## 9) Validation Plan

- Focused tests:
  - `go test ./bot`
- Broader regression tests:
  - `make integration`
  - `TEST=LuaFull make test`
  - `TEST=JSFull make test` (environment-dependent; may fail in restricted sandbox due localhost listener limits)
- Manual verification steps:
  - none in this slice.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - note `pipeline-child-rpc` now includes Go interpreter methods.
- `aidocs/COMPONENT_MAP.md` updates:
  - include `bot/pipeline_rpc_yaegi.go`.
- Connector doc updates:
  - none (no connector semantic change).
- Other docs:
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/multiprocess/ARCHITECTURE_DECISIONS.md`
