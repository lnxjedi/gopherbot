# Compatibility Note

## Change Summary

- Change:
  - Added explicit `UserMap` (`username -> internal ID`) support.
  - Kept legacy `UserRoster.UserID` mapping support with warnings.
  - Secondary protocol legacy `UserRoster` is treated as map compatibility only (attributes ignored).
  - Added protocol-scoped inbound user ID map population/use (`userIDProto`) while retaining global fallback.
- Why:
  - Separate identity mapping concerns from user directory attributes.
  - Reduce cross-protocol ambiguity in username-to-ID mapping.
- Effective date/commit:
  - 2026-02-13 (commit pending)

## What Stayed Compatible

- Unchanged behaviors:
  - Existing robots using only legacy `UserRoster.UserID` continue to map users.
  - Connector `SetUserMap` API remains unchanged.
- Unchanged config/env surfaces:
  - Existing `UserRoster` attribute fields remain valid.

## What Changed

- Behavior differences:
  - `UserMap` is now the preferred mapping source.
  - Legacy ID mapping use logs warnings.
- Startup/config/default differences:
  - Secondary `UserRoster` attributes are ignored for directory purposes; secondary legacy IDs are still read for mapping compatibility.
- Identity/routing/connector differences:
  - Inbound listed-user lookup now checks protocol-scoped ID map first, then compatibility fallback.

## Operator Actions Required

- Required config changes:
  - none immediately.
- Optional config changes:
  - migrate mapping entries from `UserRoster.UserID` to `UserMap`.
  - keep `UserRoster` for attributes only.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Deploy slice.
  2. Watch for deprecation warnings related to legacy `UserRoster.UserID`.
  3. Move IDs into `UserMap` per protocol config.
- Rollback/fallback instructions:
  - Revert this slice if needed.
- Known temporary limitations:
  - Legacy global fallback maps are still present for compatibility.

## Validation

- How to verify success:
  - confirm connectors receive expected protocol `SetUserMap` content.
  - confirm legacy robots still authenticate listed users.
- How to detect failure quickly:
  - users listed in mapping fail `IgnoreUnlistedUsers` checks.
  - wrong protocol ID selected for mapped username.

## References

- Impact report: `aidocs/multi-protocol/2026-02-13-identity-usermap-slice1/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-13-identity-usermap-slice1/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/SSH_CONNECTOR.md`
