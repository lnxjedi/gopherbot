# Compatibility Note

## Change Summary

- Change:
  - Added protocol context to selected warning/debug logs in prompt timeout/retry and related reply/subscription paths.
- Why:
  - improve diagnosability in simultaneous multi-protocol operation.
- Effective date/commit:
  - 2026-02-13 (commit pending)

## What Stayed Compatible

- Unchanged behaviors:
  - prompt matching, timeout duration, dispatch routing, and subscription semantics are unchanged.
- Unchanged config/env surfaces:
  - no config keys or env vars changed.

## What Changed

- Behavior differences:
  - none.
- Startup/config/default differences:
  - none.
- Identity/routing/connector differences:
  - none; log content only.

## Operator Actions Required

- Required config changes:
  - none.
- Optional config changes:
  - none.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Deploy slice.
  2. Verify prompt timeout/retry logs include protocol context.
- Rollback/fallback instructions:
  - revert this slice if log text changes are undesired.
- Known temporary limitations:
  - thread subscription expiration logs still cannot report protocol explicitly.

## Validation

- How to verify success:
  - trigger prompt timeout and confirm warning includes protocol.
- How to detect failure quickly:
  - no regressions expected; use test suite plus manual log check.

## References

- Impact report: `aidocs/multi-protocol/2026-02-13-log-protocol-context-slice3/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-13-log-protocol-context-slice3/pr-invariants-checklist.md`
- Related docs:
  - n/a
