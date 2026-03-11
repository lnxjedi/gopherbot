# Impact Surface Report

## 1) Change Summary

- Slice name: `identity-usermap-slice1`
- Goal: Introduce explicit per-protocol `UserMap` (`username -> transport ID`) loading and map construction, while keeping legacy `UserRoster.UserID` compatibility with warnings.
- Out of scope:
  - Removing `UserRoster.UserID` field entirely
  - Enforcing new `IgnoreUnlistedUsers` semantics (strict protocol+directory requirement)
  - Primary `conf/<protocol>.yaml` auto-load precedence redesign

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/conf.go`
  - `bot/handler.go`
  - `bot/pipecontext.go`
  - `conf/ssh.yaml`
  - `conf/terminal.yaml`
  - `robot.skel/conf/ssh.yaml`
  - `robot.skel/conf/terminal.yaml`
  - tests under `bot/` (new focused config/identity tests)
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/SSH_CONNECTOR.md`
- Key functions/types/symbols:
  - `ConfigLoader`
  - `appendRosterDataForProtocol(...)`
  - `loadConfig(preConnect bool)` user/channel map population
  - `userChanMaps` identity maps
  - `handler.IncomingMessage(...)` listed-user resolution path

## 3) Current Behavior Anchors

- Startup/order anchors:
  - `loadConfig(true)` reads primary config and initializes protocol/config maps before connector runtime.
  - `loadConfig(false)` reconciles runtime connector maps and task config.
- Routing/message-flow anchors:
  - inbound identity currently checks global `maps.userID` fallback path in `handler.IncomingMessage`.
- Identity/authorization anchors:
  - `UserRoster` currently carries both directory attributes and protocol ID mapping.
  - secondary protocol `UserRoster` entries are appended into shared roster.
- Connector behavior anchors:
  - connector user mapping is sent via `SetUserMap(...)` from runtime connector orchestration.

## 4) Proposed Behavior

- What changes:
  - Add `UserMap` config support.
  - Build per-protocol username->ID maps primarily from `UserMap`.
  - Keep legacy `UserRoster.UserID` parsing as compatibility source with warnings.
  - For secondary protocol files, parse legacy `UserRoster` as map-only compatibility path with warnings.
  - Add protocol-scoped inbound ID lookup map population (`userIDProto`) and use protocol-first match in inbound handling.
- What does not change:
  - Existing robots with only legacy `UserRoster.UserID` continue to work.
  - Connector API surface (`SetUserMap`) remains unchanged.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes (with warnings for legacy paths)
- Multi-connector isolation preserved (if applicable)?: yes

No invariant is intentionally redefined in this slice.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - identity map construction in `loadConfig` gets more explicit branches but same phase position.
- Config loading/merge/precedence impact:
  - `UserMap` and legacy `UserRoster.UserID` merge rules must be explicit and logged.
- Execution ordering impact:
  - none.
- Resource lifecycle impact:
  - connector map push path unchanged (`setConnectorUserMaps`).

## 7) Concurrency Risks

- Shared state touched:
  - `currentUCMaps.ucmap`
  - runtime connector user maps
- Locking/channel/event-order assumptions:
  - same lock boundaries as existing config replacement flow.
- Race/deadlock/starvation risks:
  - low; mostly single-threaded config load path.
- Mitigations:
  - preserve existing lock order and replace-by-swap pattern.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - no required immediate config changes.
  - legacy `UserRoster.UserID` remains functional.
- Behavior changes for operators/users:
  - warning logs for legacy mapping paths (`UserRoster.UserID` usage, secondary legacy roster map parsing).
- Migration/fallback plan:
  - migrate mapping concerns to `UserMap` incrementally, keep attributes in `UserRoster`.

## 9) Validation Plan

- Focused tests:
  - `UserMap` parsing and legacy merge precedence/warnings.
  - protocol-scoped inbound ID map usage.
- Broader regression tests:
  - `go test ./bot`
  - `go test ./...`
- Manual verification steps:
  - multi-protocol robot with overlapping usernames across protocols.
  - confirm connector mentions resolve to protocol-appropriate IDs.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - document `UserMap` + legacy `UserRoster.UserID` compatibility behavior.
- `aidocs/COMPONENT_MAP.md` updates:
  - n/a.
- Connector doc updates:
  - `aidocs/SSH_CONNECTOR.md` identity section update (`UserMap` primary, roster attrs).
- Other docs:
  - update slice artifacts checklist/compat note.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
