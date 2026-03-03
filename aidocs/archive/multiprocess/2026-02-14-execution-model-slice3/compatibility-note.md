## Change Summary

- Change:
  - External executable plugin default-config retrieval (`configure`) now runs through child runner (`gopherbot pipeline-child-exec`) instead of direct parent-process execution.
- Why:
  - Make external executable execution consistently process-isolated across runtime task execution and config-load configure path.
- Effective date/commit:
  - 2026-02-14 / execution-model-slice3 (working branch)

## What Stayed Compatible

- Unchanged behaviors:
  - `taskGo` handlers remain in-process.
  - Interpreter-backed external tasks (`.go`/`.lua`/`.js`) remain in-process.
  - Connector routing, authorization flow, and identity mapping behavior unchanged.
- Unchanged config/env surfaces:
  - No new user-facing config fields or required env vars.

## What Changed

- Behavior differences:
  - External executable plugin `configure` command execution now occurs in a short-lived child process.
- Startup/config/default differences:
  - None in startup order or config precedence; only execution mechanism changed.
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
  - Deploy slice 3 and run integration suite.
- Rollback/fallback instructions:
  - Revert this slice to restore parent-process configure execution.
- Known temporary limitations:
  - Interpreter-backed external tasks are still in-process.

## Validation

- How to verify success:
  - `go test ./bot`
  - `make integration`
- How to detect failure quickly:
  - Startup or reload failures in external plugin default config retrieval (`configure`), visible in logs and integration failures.

## References

- Impact report:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice3/impact-surface-report.md`
- PR checklist:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice3/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
