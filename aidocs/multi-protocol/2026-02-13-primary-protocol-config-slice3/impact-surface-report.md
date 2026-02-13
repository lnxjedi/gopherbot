# Impact Surface Report

## 1) Change Summary

- Slice name: `primary-protocol-config-slice3`
- Goal: Make primary protocol config loading explicit and backward compatible:
  - if `ProtocolConfig` is present in `robot.yaml`, keep using it and log compatibility warning
  - if absent, load primary protocol config from `conf/<primary>.yaml`
  - keep `UserRoster` authoritative from `robot.yaml`; treat primary protocol-file `UserRoster` as legacy mapping compatibility only
- Out of scope:
  - removing legacy `UserRoster.UserID` support
  - removing global identity fallback maps
  - changing connector runtime orchestration semantics

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/conf.go`
  - `bot/conf_protocol_test.go`
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/multi-protocol/2026-02-13-primary-protocol-config-slice3/*`
- Key functions/types/symbols:
  - `loadConfig(preConnect bool)`
  - protocol file loader helper(s) currently used for secondary protocols
  - protocol config map population (`perProtocolConfigs`)

## 3) Current Behavior Anchors

- Startup/order anchors:
  - primary protocol name resolved from `PrimaryProtocol`/`Protocol`.
  - per-protocol config map currently sets primary from `newconfig.ProtocolConfig` only.
- Routing/message-flow anchors:
  - unchanged in this slice.
- Identity/authorization anchors:
  - primary `UserMap` currently sourced from `robot.yaml` data only.
  - secondary protocol files already load `UserMap` + legacy `UserRoster` IDs as compatibility.
- Connector behavior anchors:
  - connector-specific config retrieval uses `getProtocolConfigFor(protocol)`.

## 4) Proposed Behavior

- What changes:
  - add explicit primary protocol file load path when `robot.yaml` does not include `ProtocolConfig`.
  - in compatibility mode (`ProtocolConfig` in `robot.yaml`), log warning and continue existing behavior.
  - primary protocol file contributes `ProtocolConfig`, `UserMap`, and `ChannelRoster`; primary protocol-file `UserRoster` only used for legacy ID mapping compatibility (attributes ignored).
- What does not change:
  - `ProtocolConfig` in `robot.yaml` remains valid.
  - secondary protocol loading semantics remain the same.
  - startup sequencing and connector orchestration order remain unchanged.

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
  - none beyond explicit branch for primary config source.
- Config loading/merge/precedence impact:
  - explicit precedence: `robot.yaml` `ProtocolConfig` (compat) > auto-load `conf/<primary>.yaml`.
- Execution ordering impact:
  - none.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - none.

## 7) Concurrency Risks

- Shared state touched:
  - config structures built during `loadConfig` pre/post phases.
- Locking/channel/event-order assumptions:
  - same lock/swap behavior as existing config load.
- Race/deadlock/starvation risks:
  - low.
- Mitigations:
  - no new shared mutable runtime structures.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - robots currently using include-driven `ProtocolConfig` in `robot.yaml` continue to work with warning.
  - robots that stop including protocol config can now rely on `conf/<primary>.yaml` auto-load.
- Behavior changes for operators/users:
  - warning logs in compatibility path.
- Migration/fallback plan:
  - move primary protocol config blocks out of `robot.yaml` include path into `conf/<primary>.yaml`.

## 9) Validation Plan

- Focused tests:
  - unit tests for primary protocol config source selection.
  - protocol-file map merge precedence checks.
- Broader regression tests:
  - `go test ./bot`
  - `go test ./...`
- Manual verification steps:
  - run robot with no `ProtocolConfig` in `robot.yaml` and confirm connector gets config from `conf/<primary>.yaml`.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - document explicit primary protocol config source precedence.
- `aidocs/COMPONENT_MAP.md` updates:
  - n/a.
- Connector doc updates:
  - n/a for this slice.
- Other docs:
  - add slice checklist and compatibility note.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
