# Compatibility Note

## Change Summary

- Change:
  - External executable task/plugin/job execution now runs through an internal child process command (`gopherbot pipeline-child-exec`).
  - `taskGo` and interpreter-backed external tasks (`.go`/`.lua`/`.js`) remain in-process.
- Why:
  - Introduce real process isolation in small safe scope while preserving existing behavior for complex in-process interpreter paths.
- Effective date/commit:
  - 2026-02-14 / this slice branch.

## What Stayed Compatible

- Unchanged behaviors:
  - Authorization/elevation and policy checks remain in parent pipeline flow.
  - Task ordering, pipeline semantics, and connector routing behavior remain unchanged.
  - `taskGo` remains in-process.
- Unchanged config/env surfaces:
  - No user-facing config changes.
  - No user-required environment variable changes.

## What Changed

- Behavior differences:
  - External executable tasks now run in a short-lived child gopherbot process instead of direct parent `exec.Command`.
  - `ps`/`kill` operate on child-runner pid for active external-executable runs.
- Startup/config/default differences:
  - Added internal startup fast path for `pipeline-child-exec` command (not a user-facing workflow).
- Identity/routing/connector differences:
  - None.

## Operator Actions Required

- Required config changes:
  - None.
- Optional config changes:
  - None.
- Environment variable changes:
  - None.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Deploy normally.
  2. Verify baseline behavior with integration tests.
- Rollback/fallback instructions:
  - Standard code rollback if needed.
- Known temporary limitations:
  - Interpreter-backed external tasks are intentionally still in-process in this slice.

## Validation

- How to verify success:
  - `go test ./bot`
  - `make integration`
- How to detect failure quickly:
  - Failures should appear as external task execution regressions in integration suite and pipeline logs.

## References

- Impact report:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice2/impact-surface-report.md`
- PR checklist:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice2/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/COMPONENT_MAP.md`
