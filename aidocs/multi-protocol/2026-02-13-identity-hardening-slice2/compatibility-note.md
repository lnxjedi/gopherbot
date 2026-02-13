# Compatibility Note

## Change Summary

- Change:
  - split global user directory data from per-protocol runtime identity maps.
  - kept legacy `UserRoster.UserID` compatibility as a bridge into protocol mapping.
  - preserved protocol-scoped inbound/outbound identity resolution and connector user maps.
- Why:
  - clarify identity model while keeping old robots working during migration.
- Effective date/commit:
  - 2026-02-13 (commit pending)

## What Stayed Compatible

- Unchanged behaviors:
  - `UserRoster` remains the global attribute directory.
  - `IgnoreUnlistedUsers` still requires directory + protocol mapping.
- Unchanged config/env surfaces:
  - no new keys required; existing `UserMap` key remains.

## What Changed

- Behavior differences:
  - `UserRoster` now loads as directory-only data (`DirectoryUser`), separate from protocol identity structs.
  - `UserRoster.UserID` is treated as legacy mapping input, with warnings, not as directory state.
  - explicit `UserMap` overrides legacy `UserRoster.UserID` on conflict.
- Startup/config/default differences:
  - none to phase order; identity map loading has explicit split + compatibility merge.
- Identity/routing/connector differences:
  - runtime resolution remains protocol-scoped; connector user maps are still protocol-specific.

## Operator Actions Required

- Required config changes:
  - no immediate required changes for legacy robots.
  - recommended: migrate mappings to per-protocol `UserMap` and keep attributes in global `UserRoster`.
- Optional config changes:
  - remove legacy `UserID` fields from `UserRoster` once migrated.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Deploy slice.
  2. Watch for warnings about legacy `UserRoster.UserID` compatibility.
  3. Move mappings to per-protocol `UserMap`.
- Rollback/fallback instructions:
  - revert this slice if migration is not yet complete.
- Known temporary limitations:
  - none introduced by this slice.

## Validation

- How to verify success:
  - mapped users resolve correctly per protocol.
  - users with legacy `UserRoster.UserID` still resolve IDs and generate migration warnings.
- How to detect failure quickly:
  - users unexpectedly fail mention/DM routing after config reload.

## References

- Impact report: `aidocs/multi-protocol/2026-02-13-identity-hardening-slice2/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-13-identity-hardening-slice2/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/SSH_CONNECTOR.md`
