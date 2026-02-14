# Impact Surface Report

## 1) Change Summary

- Slice name: `execution-model-slice5`
- Goal: move external Lua plugin/job/task execution and Lua plugin default-config retrieval onto the generic `pipeline-child-rpc` parent/child contract.
- Out of scope:
  - migrating external Go (`.go`) or JavaScript (`.js`) interpreters
  - changing connector behavior, identity mapping, or authorization policy placement
  - introducing long-lived child pools

## 2) Subsystems Affected (with file anchors)

- Files/directories changed:
  - `bot/calltask.go`
  - `bot/pipeline_rpc.go`
  - `bot/pipeline_rpc_lua.go` (new)
  - `modules/lua/bot_api.go` (new)
  - `modules/lua/call_extension.go`
  - `modules/lua/helpers.go`
  - `modules/lua/bot_userdata.go`
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/COMPONENT_MAP.md`
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/multiprocess/ARCHITECTURE_DECISIONS.md`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - internal child command dispatch in `bot/start.go` remains early and deterministic.
- Routing/message-flow anchors:
  - connectors still submit to parent engine; parent keeps pipeline/routing authority.
- Identity/authorization anchors:
  - policy and identity resolution remain in parent flows.

## 4) Proposed Behavior

- What changes:
  - Lua runtime calls now execute in a child `pipeline-child-rpc` process (`lua_run`).
  - Lua plugin default config retrieval now executes over RPC (`lua_get_config`).
  - child uses a narrow Lua `BotAPI` proxy; parent serves Robot API calls via `robot_call`.
- What does not change:
  - compiled-in `taskGo`/`bot/*` remains in-process.
  - external `.go` and `.js` interpreter paths remain in-process in this slice.
  - connector and identity behavior are unchanged.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes
- Multi-connector isolation preserved?: yes

No invariant redefinition is proposed.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none beyond existing internal command path.
- Config loading/merge impact:
  - Lua plugin `configure` now retrieved through RPC child path.
- Execution ordering impact:
  - parent/child request-response with correlation IDs, including parent-served robot calls.

## 7) Concurrency Risks

- Parent waits for child responses and may service robot calls while awaiting target response.
- Risk: protocol deadlock if request/response ordering is wrong.
- Mitigation: strict message IDs, single-request loop per child invocation, explicit shutdown path.

## 8) Backward Compatibility

- Existing Lua extension API behavior is preserved; migration is internal execution-model only.
- No user-facing config changes introduced.

## 9) Validation Plan

- Focused:
  - `go test ./bot`
- Broad:
  - `make integration`
  - `RUN_FULL=lua make test`
- Expected gate:
  - `TestLuaFull` passes with config + HTTP assertions.

## 10) Documentation Plan

- Update startup/component/execution-model docs to reflect active Lua-over-RPC routing.
- Record slice decision in multiprocess architecture notes.
