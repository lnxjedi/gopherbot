# Impact Surface Report

## 1) Change Summary

- Slice name: `protocol-send-user-channel-slice3`
- Goal: add `SendProtocolUserChannelMessage` to the Robot API and extension libraries to support explicit cross-protocol sends with user/channel semantics.
- Out of scope:
  - protocol-agnostic shared profile store
  - cross-protocol thread mapping
  - changing existing default send routing for `Say`/`Reply`/`SendUser*`

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `robot/robot.go`
  - `bot/send_message.go`
  - `bot/http.go`
  - `bot/connector_runtime_test.go`
  - `modules/javascript/*`
  - `modules/lua/*`
  - `modules/yaegi-dynamic-go/yaegi_symbols.go`
  - `lib/gopherbot_v1.*`, `lib/gopherbot_v2.py`, `lib/GopherbotV1.jl`
  - `aidocs/EXTENSION_API.md`
- Key functions/types/symbols:
  - `robot.Robot` interface in `robot/robot.go`
  - send helpers in `bot/send_message.go`
  - HTTP dispatch in `bot/http.go`
  - runtime connector lookup helpers in `bot/connector_runtime.go`
  - language bindings in `modules/javascript/send_messages.go` and `modules/lua/send_messages.go`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - unchanged; runtime manager startup and connector init are already in `bot/start.go` + `bot/bot_process.go`.
- Routing/message-flow anchors:
  - outbound sends route by `msgObject.Protocol` through `runtimeConnectorRouter` in `bot/connector_runtime.go`.
  - existing methods are channel-thread, user-channel-thread, and DM only.
- Identity/authorization anchors:
  - permission checks are username-based in engine flow (unchanged).
  - user lookup uses roster-backed maps and protocol-local IDs.
- Connector behavior anchors:
  - connectors implement `SendProtocolChannelThreadMessage`, `SendProtocolUserChannelThreadMessage`, `SendProtocolUserMessage`.

## 4) Proposed Behavior

- What changes:
  - add `SendProtocolUserChannelMessage(protocol, user, channel, message, ...)`.
  - semantics:
    - `user != "" && channel == ""` => DM
    - `channel != "" && user == ""` => channel message
    - `user != "" && channel != ""` => user-in-channel message
    - `user == "" && channel == ""` => missing-arguments failure
  - expose method in HTTP bridge and all shipped extension libraries/interpreter bindings.
- What does not change:
  - existing send APIs and behaviors stay intact.
  - startup/config/load order unchanged.
  - auth policy location unchanged (still engine-managed).

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes
- Multi-connector isolation preserved (if applicable)?: yes

No invariant redefinition required.

## 6) Cross-Cutting Concerns

- Startup sequencing impact: none.
- Config loading/merge/precedence impact: none.
- Execution ordering impact: none; send dispatch remains synchronous per call path.
- Resource lifecycle impact (connections, goroutines, shutdown): none.

## 7) Concurrency Risks

- Shared state touched:
  - runtime connector map read paths for protocol send dispatch.
  - user/channel map reads for protocol-specific ID resolution.
- Locking/channel/event-order assumptions:
  - existing runtime connector locks remain unchanged.
- Race/deadlock/starvation risks:
  - low; primarily additional read-only paths.
- Mitigations:
  - reuse existing resolver helpers and connector router APIs.
  - add focused tests around routing/semantics.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - no config changes required.
  - existing plugins/jobs keep current behavior.
- Behavior changes for operators/users:
  - new optional API for extension authors.
- Migration/fallback plan:
  - continue using existing send methods; new method is additive.

## 9) Validation Plan

- Focused tests:
  - `bot` package tests for protocol send semantics (`dm`/`channel`/`user+channel`/missing args).
- Broader regression tests:
  - `go test ./bot ./connectors/... ./modules/...`
- Manual verification steps:
  - from one protocol, call new API to send to another protocol user/channel.
  - verify send target behavior for empty user or empty channel cases.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates: none expected.
- `aidocs/COMPONENT_MAP.md` updates: none expected.
- Connector doc updates: none expected (send semantics unchanged at connector level).
- Other docs:
  - update `aidocs/EXTENSION_API.md`.
  - include this slice artifacts under `aidocs/multi-protocol/...`.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
