# Impact Surface Report

## 1) Change Summary

- Slice name: `channel-maps-protocol-slice4`
- Goal: make channel-name and channel-ID resolution protocol-scoped first, with safe global fallback for compatibility.
- Out of scope:
  - user-map behavior changes
  - protocol-agnostic shared identity/profile store design
  - thread mapping across protocols

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/conf.go`
  - `bot/send_message.go`
  - `bot/handler.go`
  - `bot/pipecontext.go`
  - `bot/connector_runtime_test.go`
  - `aidocs/multi-protocol/2026-02-12-protocol-send-user-channel-slice3/pr-invariants-checklist.md` (residual-risk update)
- Key functions/types/symbols:
  - `userChanMaps` in `bot/conf.go`
  - channel roster loading in `loadConfig(...)` (`bot/conf.go`)
  - send channel resolution helpers in `bot/send_message.go`
  - inbound channel-ID mapping in `handler.IncomingMessage(...)` (`bot/handler.go`)
  - `worker.registerActive(...)` channel resolution in `bot/pipecontext.go`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - unchanged; config load still builds `currentUCMaps` pre-runtime.
- Routing/message-flow anchors:
  - outbound channel resolution currently uses global `maps.channel`.
  - inbound `ChannelID -> ChannelName` currently uses global `maps.channelID`.
- Identity/authorization anchors:
  - authorization remains username-based and unchanged.
- Connector behavior anchors:
  - connectors receive resolved IDs via existing send interfaces; no connector API changes required.

## 4) Proposed Behavior

- What changes:
  - `userChanMaps` gains protocol-scoped channel maps (by name and by ID).
  - outbound channel resolution uses protocol-specific map first.
  - inbound channel-ID resolution uses protocol-specific map first.
  - global channel maps remain as fallback for compatibility when protocol-specific entry is unavailable.
- What does not change:
  - connector method signatures and send flow.
  - startup sequence, config precedence, and connector lifecycle.
  - permission model and policy placement.

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
- Config loading/merge/precedence impact: channel roster merge now fills protocol-scoped maps in addition to legacy globals.
- Execution ordering impact: none.
- Resource lifecycle impact (connections, goroutines, shutdown): none.

## 7) Concurrency Risks

- Shared state touched:
  - `currentUCMaps.ucmap` structure content.
- Locking/channel/event-order assumptions:
  - map replacement remains atomic under existing `currentUCMaps` mutex.
- Race/deadlock/starvation risks:
  - low; read-path map lookups only.
- Mitigations:
  - preserve existing lock boundaries and fallback behavior.
  - add focused tests for duplicate channel names across protocols.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - no config schema change required.
  - behavior becomes more correct when channel names overlap across protocols.
- Behavior changes for operators/users:
  - cross-protocol targeted sends resolve channel names to the correct protocol-specific channel ID.
- Migration/fallback plan:
  - none required; fallback to global map preserved.

## 9) Validation Plan

- Focused tests:
  - extend `bot/connector_runtime_test.go` to verify duplicate channel names resolve by target protocol.
  - verify `SendProtocolUserChannelMessage(...)` still passes all prior semantics.
- Broader regression tests:
  - `go test ./bot ./connectors/... ./modules/...`
  - `go test ./...`
- Manual verification steps:
  - configure same channel name on ssh+slack with different IDs.
  - run cross-protocol send and confirm target connector receives expected protocol-specific ID.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates: none expected.
- `aidocs/COMPONENT_MAP.md` updates: none expected.
- Connector doc updates: none expected (engine map behavior only).
- Other docs:
  - update slice checklist residual-risk note now that channel map risk is resolved.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
