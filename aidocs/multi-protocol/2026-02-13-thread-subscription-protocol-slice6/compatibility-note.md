# Compatibility Note

## Change Summary

- Change:
  - thread subscription keys now include protocol.
  - legacy persisted subscription keys without protocol are still supported on restore.
  - config loader accepts legacy `UserRoster.UserID` fields for backward-compatible parsing.
- Why:
  - prevent cross-protocol subscription collisions and keep old robots/tests working.

## What Stayed Compatible

- Legacy subscription memory entries continue to restore.
- Legacy configs with `UserRoster.UserID` still load.
- Existing Subscribe/Unsubscribe plugin APIs are unchanged.

## What Changed

- Subscription matching is protocol-scoped (`protocol + channel + thread`).
- Expiration/debug logs can include protocol where available.

## Operator Actions Required

- Required config changes:
  - none.
- Optional config changes:
  - migrate `UserRoster.UserID` to per-protocol `UserMap`.

## Rollout / Fallback

- Recommended rollout:
  1. Deploy change.
  2. Reload robot.
  3. Verify subscription behavior in two protocols with same channel/thread labels.
- Rollback:
  - revert slice if unexpected subscription routing behavior appears.

## Validation

- Verify protocol-scoped thread subscriptions do not leak across connectors.
- Run `make test` and inspect `/tmp/bottest.log` on failures.

## References

- Impact report: `aidocs/multi-protocol/2026-02-13-thread-subscription-protocol-slice6/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-13-thread-subscription-protocol-slice6/pr-invariants-checklist.md`
