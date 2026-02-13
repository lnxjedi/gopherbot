# Impact Surface Report

## 1) Change Summary

- Slice name: `identity-ignoreunlisted-slice2`
- Goal: Enforce strict `IgnoreUnlistedUsers` semantics requiring both (a) presence in global user directory (`UserRoster`) and (b) membership in protocol-specific identity mapping (`UserMap`/legacy mapping), without changing default behavior when `IgnoreUnlistedUsers` is false.
- Out of scope:
  - Removing legacy `UserRoster.UserID` compatibility
  - Reworking primary protocol config auto-load precedence
  - Connector API changes

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/conf.go`
  - `bot/handler.go`
  - tests under `bot/`
  - `aidocs/STARTUP_FLOW.md`
  - slice artifacts in `aidocs/multi-protocol/2026-02-13-identity-ignoreunlisted-slice2/`
- Key functions/types/symbols:
  - `userChanMaps`
  - `loadConfig(preConnect bool)` map population
  - `handler.IncomingMessage(...)` listed-user evaluation and ignore gate

## 3) Current Behavior Anchors

- Startup/order anchors:
  - user/channel maps are built in `loadConfig` before runtime handoff.
- Routing/message-flow anchors:
  - inbound processing in `IncomingMessage` sets `listedUser` from ID mapping lookup.
- Identity/authorization anchors:
  - protocol-first `userIDProto` lookup exists, but synthetic user entries from `UserMap` can satisfy listed-user checks even if not present in attribute roster.
- Connector behavior anchors:
  - connectors receive protocol user maps via `setConnectorUserMaps(...)`.

## 4) Proposed Behavior

- What changes:
  - Introduce explicit global directory membership set derived only from `UserRoster` usernames.
  - Track whether inbound messages matched protocol-specific mapping.
  - For `IgnoreUnlistedUsers=true`, accept messages only when both are true:
    - username exists in global directory membership set
    - inbound ID resolved through protocol-specific mapping (`userIDProto`)
- What does not change:
  - `IgnoreUnlistedUsers=false` behavior remains permissive.
  - Legacy mapping compatibility remains.
  - Connector map push and startup ordering remain unchanged.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes
- Multi-connector isolation preserved (if applicable)?: yes

No invariant is intentionally redefined.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none; additional map set built in same config-load phase.
- Config loading/merge/precedence impact:
  - none to precedence; only consumption semantics tighten for `IgnoreUnlistedUsers`.
- Execution ordering impact:
  - none.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - none.

## 7) Concurrency Risks

- Shared state touched:
  - `currentUCMaps.ucmap`
- Locking/channel/event-order assumptions:
  - preserve existing replace-by-swap behavior under lock.
- Race/deadlock/starvation risks:
  - low.
- Mitigations:
  - no new locks; extend existing immutable map snapshot struct.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - robots with `IgnoreUnlistedUsers=true` and users present only in protocol `UserMap` (not global `UserRoster`) will now be rejected.
- Behavior changes for operators/users:
  - stricter listed-user gate under `IgnoreUnlistedUsers`.
- Migration/fallback plan:
  - add users to global `UserRoster` directory while keeping per-protocol `UserMap` entries.

## 9) Validation Plan

- Focused tests:
  - unit tests for strict listed-user gate (directory + protocol map).
  - positive/negative cases for protocol mapping vs global fallback.
- Broader regression tests:
  - `go test ./bot`
  - `go test ./...`
- Manual verification steps:
  - multi-protocol robot, `IgnoreUnlistedUsers=true`, verify mapped-but-undirectory users are ignored.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - clarify strict `IgnoreUnlistedUsers` requirement in identity mapping section.
- `aidocs/COMPONENT_MAP.md` updates:
  - n/a.
- Connector doc updates:
  - n/a for this slice.
- Other docs:
  - add PR checklist + compatibility note for this slice.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
