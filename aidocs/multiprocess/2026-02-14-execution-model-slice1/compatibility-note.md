# Compatibility Note

## Change Summary

- Change:
  - Added an explicit task execution boundary in pipeline runtime: `runPipeline -> executeTask -> callTask`.
  - Added runner-selection scaffolding in `bot/task_execution.go`.
  - Encoded invariant: compiled-in `taskGo` (`bot/*`) execution remains in-process.
- Why:
  - Establish a safe seam for future multiprocess execution slices without changing current behavior.
- Effective date/commit:
  - 2026-02-14 / this slice branch.

## What Stayed Compatible

- Unchanged behaviors:
  - Pipeline ordering, authorization, elevation, and reply behavior.
  - Startup mode detection and startup sequencing.
  - Connector routing and identity resolution behavior.
- Unchanged config/env surfaces:
  - No new config keys.
  - No new environment variables.

## What Changed

- Behavior differences:
  - None expected externally; this is internal execution-path refactoring only.
- Startup/config/default differences:
  - None.
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
  1. Deploy as normal.
  2. Verify baseline behavior via existing integration tests.
- Rollback/fallback instructions:
  - Standard code rollback if needed.
- Known temporary limitations:
  - Multiprocess task execution is not enabled in this slice; only the execution boundary exists.

## Validation

- How to verify success:
  - `go test ./bot`
  - `make integration`
- How to detect failure quickly:
  - Regressions would appear as task dispatch/pipeline execution failures in existing test suites.

## References

- Impact report:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice1/impact-surface-report.md`
- PR checklist:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice1/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/PIPELINE_LIFECYCLE.md`
  - `aidocs/COMPONENT_MAP.md`
