# Impact Surface Report

## Phase 0 Summary (Completed)

- Core invariants: startup and config are deterministic; auth/policy decisions are engine-level; identity mapping must be deterministic.
- Startup order: `main.go` -> `bot.Start()` (`bot/start.go`) -> `initBot()` -> `run()` (`bot/bot_process.go`) -> connector loop starts before post-connect `loadConfig(false)`.
- Connector assumptions: current runtime selects one connector via `currentCfg.protocol` in `bot/start.go`.
- Message routing model: incoming messages are normalized in `handler.IncomingMessage` (`bot/handler.go`) and dispatched through engine plugin/job logic.
- Identity model: roster mapping in `bot/conf.go` builds `userID -> username` and `username -> user` maps used by `handler.IncomingMessage`, admin checks, and authorization visibility logic.

## 1) Change Summary

- Slice name: `config-identity-slice1`
- Goal: introduce multi-protocol-ready config and identity foundations without introducing full simultaneous connector runtime.
- Out of scope:
  - running multiple connectors concurrently
  - connector lifecycle admin commands (start/stop secondaries)
  - cross-protocol send API additions (`SendProtocolUserChannelMessage`)
  - thread-mapping behavior across protocols

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/conf.go`
  - `bot/bot_process.go`
  - `bot/start.go`
  - `conf/robot.yaml`
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/COMPONENT_MAP.md`
  - `aidocs/README.md` (already updated for robot operational model)
  - `test/` config/startup integration coverage as needed
- Key functions/types/symbols:
  - `type ConfigLoader` (`bot/conf.go`)
  - `loadConfig(preConnect bool)` (`bot/conf.go`)
  - `type configuration` (`bot/bot_process.go`)
  - connector selection in `Start(...)` (`bot/start.go`)
  - `handler.IncomingMessage` (`bot/handler.go`)
  - `worker.checkAdmin` (`bot/robot.go`)
  - `getProtocol` (`bot/util.go`)

## 3) Current Behavior Anchors

- Startup/order anchors:
  - startup mode detection in `detectStartupMode` (`bot/config_load.go`)
  - single connector initialization in `bot/start.go` (`initializeConnector := connectors[currentCfg.protocol]`)
  - single connector loop in `run()` (`bot/bot_process.go`)
- Routing/message-flow anchors:
  - message normalization and listed-user detection in `handler.IncomingMessage` (`bot/handler.go`)
  - plugin dispatch and unlisted-user ambient behavior in `bot/dispatch.go`
- Identity/authorization anchors:
  - global roster map construction in `loadConfig` (`bot/conf.go`)
  - admin checks in `worker.checkAdmin` (`bot/robot.go`)
  - `IgnoreUnlistedUsers` drop behavior in `handler.IncomingMessage`
- Connector behavior anchors:
  - connector interface expects one active engine connector (`robot/connector_defs.go`)
  - connector user mapping via `SetUserMap(map[string]string)` (all connector implementations)

## 4) Proposed Behavior

- What changes:
  - Add `PrimaryProtocol` and `SecondaryProtocols` config keys.
  - Keep `Protocol` for backward compatibility; if both `PrimaryProtocol` and `Protocol` are present and differ, prefer `PrimaryProtocol` and emit a warning.
  - Establish identity model where transport-specific IDs map to exact-case usernames; reject uppercase usernames in rosters (lowercase letters and digits allowed).
  - Define attribute resolution policy for duplicate username across protocols as source-protocol first.
  - Preserve `IgnoreUnlistedUsers` semantics:
    - true: drop unlisted users early
    - false: allow non-privileged behavior but fail admin/group checks for unlisted users
  - Keep per-connector roster in connector config files (`conf/<connector>.yaml` in robot custom repo).
- What does not change:
  - single-connector runtime behavior in this slice
  - default reply behavior (same protocol as source message)
  - job scheduling semantics except documented future plan (cron sends to primary protocol once runtime fan-out exists)

## 5) Invariant Impact Check

- Startup determinism preserved?: yes (config parsing changes only in this slice)
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes (no connector runtime change in this slice)
- Config precedence still explicit?: yes (`conf/robot.yaml` + merge order unchanged)
- Multi-connector isolation preserved (if applicable)?: n/a in this slice (no simultaneous run loop yet)

No invariant redefinition in this slice.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - pre-connect config parsing expands to include primary/secondary protocol fields and compatibility alias handling.
- Config loading/merge/precedence impact:
  - defaults remain installed first, custom override second (`getConfigFile` merge in `bot/config_load.go`)
  - connector-specific includes remain template-driven in robot config.
- Execution ordering impact:
  - none in this slice; connector startup flow remains single-path.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - none in this slice; no additional connector goroutines introduced.

## 7) Concurrency Risks

- Shared state touched:
  - config snapshot (`currentCfg.configuration`)
  - user/channel mapping snapshot (`currentUCMaps.ucmap`)
- Locking/channel/event-order assumptions:
  - maintain existing "replace pointer snapshot" pattern under lock to avoid partial map visibility.
- Race/deadlock/starvation risks:
  - low in this slice if no new shared mutable maps are introduced without snapshot replacement.
- Mitigations:
  - keep pre-connect/post-connect update model and lock boundaries unchanged
  - avoid in-place mutation of maps read by workers

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - existing `Protocol:` remains valid.
  - no required immediate migration for existing single-protocol robots.
- Behavior changes for operators/users:
  - if both `PrimaryProtocol` and `Protocol` are set and differ, warning logged and `PrimaryProtocol` wins.
  - uppercase username entries will be rejected by new validation.
- Migration/fallback plan:
  - encourage moving to `PrimaryProtocol` + `SecondaryProtocols`.
  - keep `Protocol` compatibility path until a later deprecation phase.

## 9) Validation Plan

- Focused tests:
  - config parse tests for:
    - `Protocol` only
    - `PrimaryProtocol` only
    - both set equal
    - both set conflicting (warning + `PrimaryProtocol` chosen)
    - `SecondaryProtocols` parse/store behavior
  - roster validation tests for lowercase/digit usernames and uppercase rejection
  - `IgnoreUnlistedUsers` behavior regression checks
- Broader regression tests:
  - `make test` integration suite
  - targeted connector tests affected by config loader fields
- Manual verification steps:
  - run a local robot with only `Protocol` configured and verify no behavior change
  - run with both `PrimaryProtocol` and `Protocol` conflicting and confirm warning/selection behavior

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - document `PrimaryProtocol`/`SecondaryProtocols` semantics once implemented.
- `aidocs/COMPONENT_MAP.md` updates:
  - update if config/runtime ownership boundaries move.
- Connector doc updates:
  - `aidocs/SSH_CONNECTOR.md` and Slack docs as multi-protocol runtime phases land.
- Other docs:
  - maintain `aidocs/multi-protocol/*` per-slice reports/checklists/compat notes.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
