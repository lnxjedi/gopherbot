# Impact Surface Report

## 1) Change Summary

- Slice name: `execution-model-slice6`
- Goal: move external JavaScript plugin/job/task execution and JavaScript plugin default-config retrieval onto the generic `pipeline-child-rpc` contract.
- Out of scope:
  - migrating external Go (`.go` / yaegi) interpreter execution
  - changing connector, identity, or authorization semantics
  - changing startup mode selection behavior

## 2) Subsystems Affected (with file anchors)

- `bot/calltask.go`
- `bot/pipeline_rpc.go`
- `bot/pipeline_rpc_javascript.go` (new)
- `modules/javascript/bot_api.go` (new)
- `modules/javascript/call_extension.go`
- `modules/javascript/bot_object.go`
- `aidocs/STARTUP_FLOW.md`
- `aidocs/COMPONENT_MAP.md`
- `aidocs/EXECUTION_SECURITY_MODEL.md`
- `aidocs/multiprocess/ARCHITECTURE_DECISIONS.md`

## 3) Current Behavior Anchors

- Startup fast-path internal child commands remain parsed in `Start(...)`.
- Parent engine remains routing/policy/identity authority.
- Existing Lua-over-RPC flow remains intact.

## 4) Proposed Behavior

- What changes:
  - JavaScript execution path now uses `js_run` over child RPC.
  - JavaScript plugin configure/default-config path now uses `js_get_config` over child RPC.
  - JavaScript runtime now targets a narrow `BotAPI` interface to support RPC-backed clients.
- What does not change:
  - external Go interpreter path remains in-process.
  - compiled-in `taskGo` behavior remains in-process.
  - connector/runtime semantics remain unchanged.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Auth/policy in engine flows?: yes
- Username-based permission semantics unchanged?: yes
- Connector message ordering unchanged?: yes
- Multi-connector isolation unchanged?: yes

## 6) Cross-Cutting Concerns

- Parent/child protocol sequencing and request correlation IDs must remain deterministic.
- Child error propagation must preserve clear mechanism-fail behavior.
- JS configure behavior must remain consistent for plugin default config loading.

## 7) Concurrency Risks

- Parent must continue servicing `robot_call` requests while awaiting target response.
- Risk of protocol deadlock mitigated by strict single-request lifecycle and explicit shutdown.

## 8) Backward Compatibility

- No extension-facing API syntax changes expected.
- No user/operator config migration required for this slice.

## 9) Validation Plan

- `go test ./bot`
- `make integration`
- `RUN_FULL=js make test`

## 10) Documentation Plan

- Update startup/component/execution-model docs for JS RPC status.
- Add slice-specific compatibility and invariants artifacts.
