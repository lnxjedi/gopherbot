# Compatibility Note

## Change Summary

- Change:
  - `IgnoreUnlistedUsers` now requires two conditions for inbound acceptance:
    - user exists in global `UserRoster` directory
    - inbound ID maps via protocol-specific identity mapping
  - added explicit global directory-membership set derived from `UserRoster`.
- Why:
  - enforce deterministic and explicit identity policy in multi-protocol runtime.
- Effective date/commit:
  - 2026-02-13 (commit pending)

## What Stayed Compatible

- Unchanged behaviors:
  - with `IgnoreUnlistedUsers: false`, inbound behavior remains permissive.
  - legacy `UserRoster.UserID` mapping compatibility remains in place.
- Unchanged config/env surfaces:
  - no new config keys required.

## What Changed

- Behavior differences:
  - users that are only in protocol `UserMap` (and missing from global `UserRoster`) are now treated as unlisted when `IgnoreUnlistedUsers` is true.
  - users that resolve only through global fallback ID mapping are now treated as unlisted when `IgnoreUnlistedUsers` is true.
- Startup/config/default differences:
  - none.
- Identity/routing/connector differences:
  - `listedUser` now means both protocol mapping + global directory membership.

## Operator Actions Required

- Required config changes:
  - for robots using `IgnoreUnlistedUsers: true`, ensure each allowed user is present in global `UserRoster` and in each relevant protocol `UserMap`.
- Optional config changes:
  - none.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Deploy and monitor for IgnoreUnlistedUsers debug logs.
  2. Add missing users to global `UserRoster` where needed.
- Rollback/fallback instructions:
  - revert this slice if strict gate behavior is not yet ready.
- Known temporary limitations:
  - global fallback identity map is still present for compatibility name resolution.

## Validation

- How to verify success:
  - with `IgnoreUnlistedUsers: true`, confirm messages are accepted only when directory + protocol mapping both exist.
- How to detect failure quickly:
  - expected users are ignored with debug log showing gate details.

## References

- Impact report: `aidocs/multi-protocol/2026-02-13-identity-ignoreunlisted-slice2/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-13-identity-ignoreunlisted-slice2/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/STARTUP_FLOW.md`
