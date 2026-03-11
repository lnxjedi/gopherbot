# Compatibility Note

## Change Summary

- Change:
  - Added multi-protocol runtime orchestration with primary + secondary connector lifecycle management.
  - Added built-in admin commands for runtime protocol control: `protocol-list`, `protocol-start`, `protocol-stop`, `protocol-restart` (space or hyphen form).
  - Added reload reconciliation for `SecondaryProtocols` and explicit primary-change-on-reload rejection.
- Why:
  - Enable simultaneous connector operation while preserving primary protocol control and startup determinism.
- Effective date/commit:
  - 2026-02-12 / this slice branch.

## What Stayed Compatible

- Unchanged behaviors:
  - Primary startup failure remains fatal.
  - Scheduled jobs and cron-triggered sends still use primary protocol by default.
  - Existing admin commands (`reload`, `quit`, `restart`, etc.) remain unchanged.
- Unchanged config/env surfaces:
  - Legacy `Protocol` key remains accepted.
  - Existing `ProtocolConfig` loading for primary remains valid.

## What Changed

- Behavior differences:
  - Startup attempts all configured protocols (primary required, secondaries best effort with logged failures).
  - Secondary connector exit/failure does not terminate whole robot.
  - Connector send routing is now protocol-aware by source message protocol.
- Startup/config/default differences:
  - `SecondaryProtocols` now actively affects runtime lifecycle.
  - Removing a protocol from `SecondaryProtocols` on reload stops it.
  - Reload retry behavior for failed secondaries occurs only on reload/admin command (no background timer retry).
- Identity/routing/connector differences:
  - Secondary connector rosters are loaded from `conf/<protocol>.yaml` for merged user/channel mapping.
  - Per-protocol user maps are applied to connectors where available.

## Operator Actions Required

- Required config changes:
  - None for single-protocol robots.
- Optional config changes:
  - Add `PrimaryProtocol` explicitly (preferred modern key).
  - Add `SecondaryProtocols` and per-protocol config files (`conf/<protocol>.yaml`) to enable simultaneous connectors.
- Environment variable changes:
  - None required for this slice.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Set `PrimaryProtocol` (or keep existing `Protocol`) and verify baseline startup.
  2. Add one protocol to `SecondaryProtocols` with matching `conf/<protocol>.yaml`.
  3. Reload and verify status with `protocol-list`.
  4. Add remaining secondaries incrementally.
- Rollback/fallback instructions:
  - Remove entries from `SecondaryProtocols` and reload.
  - If needed, revert to single protocol by keeping only primary protocol config.
- Known temporary limitations:
  - SSH connector does not currently disconnect active sessions when a roster entry is removed on reload.

## Validation

- How to verify success:
  - Startup logs show initialization attempts for primary and configured secondaries.
  - `protocol-list` reports running/failure status.
  - Replies stay on source protocol for connector-originated commands.
- How to detect failure quickly:
  - Secondary init/start failures show explicit log errors and `protocol-list` state `failed`.
  - Primary reload change attempts log rejection and continue on active primary.

## References

- Impact report:
  - `aidocs/multi-protocol/2026-02-12-runtime-orchestration-slice2/impact-surface-report.md`
- PR checklist:
  - `aidocs/multi-protocol/2026-02-12-runtime-orchestration-slice2/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/COMPONENT_MAP.md`
  - `aidocs/TESTING_CURRENT.md`
