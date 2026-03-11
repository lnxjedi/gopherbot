# Impact Surface Report

## 1) Change Summary

- Slice name: `identity-hardening-slice2` (struct split + compatibility bridge)
- Goal:
  - split global user directory data from protocol identity mapping
  - make global roster (`UserRoster`) attribute-only (no runtime dependency on `UserID`)
  - keep backward compatibility by translating legacy `UserRoster.UserID` into protocol user maps when needed
- Out of scope:
  - connector auth redesign
  - thread model redesign
  - permission policy behavior changes beyond identity source wiring

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/conf.go`
  - `bot/robot_connector_methods.go`
  - `bot/conf_protocol_test.go`
  - `bot/conf_identity_test.go`
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/SSH_CONNECTOR.md`
  - `aidocs/multi-protocol/2026-02-13-identity-hardening-slice2/*`
- Key functions/types/symbols:
  - `ConfigLoader`
  - `DirectoryUser`
  - `UserInfo`
  - `loadConfig(preConnect bool)`
  - `loadProtocolFileData(...)`
  - `userChanMaps`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - identity maps are built during `loadConfig` before runtime usage.
- Identity anchors:
  - mixed usage of `UserInfo` currently conflates directory attributes and protocol IDs.
  - legacy `UserRoster.UserID` is still used for compatibility in some flows.
- Config precedence anchors:
  - primary protocol config may come from `robot.yaml` compatibility path or `conf/<primary>.yaml`.

## 4) Proposed Behavior

- What changes:
  - `UserRoster` loads into `DirectoryUser` entries (global attribute directory only).
  - per-protocol runtime identity maps are built from `UserMap`.
  - legacy `UserRoster.UserID` values are parsed only for compatibility mapping population and warning/logged migration guidance.
  - if `UserMap` and legacy IDs both exist, `UserMap` wins on conflict.
- What does not change:
  - authorization decisions still use protocol-agnostic usernames.
  - startup ordering and connector isolation behavior remain unchanged.
  - channel roster behavior remains protocol-scoped.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes
- Multi-connector isolation preserved?: yes

No invariant is intentionally redefined.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none to phase ordering.
- Config loading/merge/precedence impact:
  - explicit split between directory and mapping sources.
  - compatibility bridge must preserve old robot behavior with warnings.
- Execution ordering impact:
  - none.
- Runtime lifecycle impact:
  - none expected.

## 7) Concurrency Risks

- Shared state touched:
  - `currentUCMaps.ucmap`
- Locking assumptions:
  - continue lock/swap discipline; build maps locally, then swap once.
- Risk level:
  - low.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - old robots with `UserRoster.UserID` continue to work through compatibility mapping.
  - warnings prompt migration to `UserMap`.
- Behavior changes for operators/users:
  - clearer model: global `UserRoster` = directory, per-protocol `UserMap` = identity mapping.
- Migration guidance:
  - move IDs to per-protocol `UserMap`; keep shared attributes in main `UserRoster`.

## 9) Validation Plan

- Focused tests:
  - compatibility mapping tests (`UserRoster.UserID` -> protocol map)
  - conflict precedence tests (`UserMap` overrides legacy IDs)
  - directory-only behavior tests
- Broader regression tests:
  - `go test ./bot`
  - `go test ./...`

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - document split model and compatibility bridge details.
- `aidocs/SSH_CONNECTOR.md` updates:
  - identity mapping references to `UserMap` plus legacy warning behavior.
- Other docs:
  - compatibility note + checklist updates under slice docs.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
