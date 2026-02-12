# Compatibility Note

## Change Summary

- Change:
  - Added protocol-scoped channel lookup maps (channel-name and channel-ID) in engine config state.
  - Updated inbound and outbound channel resolution paths to prefer protocol-scoped lookup, with legacy global fallback preserved.
- Why:
  - Prevent cross-protocol channel-name/ID collisions from resolving to the wrong connector target.
- Effective date/commit:
  - 2026-02-12 / this slice branch.

## What Stayed Compatible

- Unchanged behaviors:
  - Existing config keys and connector setup workflow are unchanged.
  - Existing robots without duplicate channel names across protocols continue to behave as before.
  - Connector APIs and plugin send methods are unchanged.
- Unchanged config/env surfaces:
  - No new config keys or env vars.
  - Existing `ChannelRoster` layout remains the same.

## What Changed

- Behavior differences:
  - Channel resolution now uses protocol-specific maps first.
  - `SendProtocolUserChannelMessage` channel target now resolves against target protocol before global fallback.
  - Incoming `ChannelID -> ChannelName` mapping now resolves by incoming protocol before global fallback.
- Startup/config/default differences:
  - none.
- Identity/routing/connector differences:
  - Routing correctness improves when multiple protocols share a channel name (for example `general`).
  - No identity model change in this slice.

## Operator Actions Required

- Required config changes:
  - none.
- Optional config changes:
  - none.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Pull updated engine.
  2. Reload robot config.
  3. Validate cross-protocol sends for any duplicated channel names.
- Rollback/fallback instructions:
  - Roll back engine to previous commit if needed; no config rollback required.
- Known temporary limitations:
  - Protocol-unknown lookup paths still rely on legacy global fallback.

## Validation

- How to verify success:
  - Configure duplicated channel name across two protocols with distinct channel IDs.
  - Send to each protocol and verify connector receives its protocol-specific channel ID.
  - Verify inbound messages still resolve channel name correctly for each protocol.
- How to detect failure quickly:
  - Cross-protocol send lands in wrong channel ID despite correct target protocol.
  - Logs show fallback lookups in protocol-known paths where protocol-scoped mapping should exist.

## References

- Impact report:
  - `aidocs/multi-protocol/2026-02-12-channel-maps-protocol-slice4/impact-surface-report.md`
- PR checklist:
  - `aidocs/multi-protocol/2026-02-12-channel-maps-protocol-slice4/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/multi-protocol/2026-02-12-protocol-send-user-channel-slice3/pr-invariants-checklist.md`
