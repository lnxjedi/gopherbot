## Change Summary

- Change:
  - RPC interpreter execution now tracks child PID/cancel handle in worker state.
  - Admin `kill` now cancels RPC request state and interrupts targeted prompt waiters.
  - RPC lifecycle uses bounded hello/request/shutdown/exit waits with explicit error classes.
- Why:
  - close operational gaps where prompt-blocked interpreter tasks could not be terminated via admin controls.
- Effective date/commit:
  - 2026-02-14 / execution-model-slice8 (working branch)

## What Stayed Compatible

- Unchanged behaviors:
  - compiled-in `taskGo`/`bot/*` handlers remain in-process.
  - connector routing, identity mapping, and authorization behavior remain unchanged.
  - external executable child-runner path remains unchanged.
- Unchanged config/env surfaces:
  - no new required config keys or environment variables.

## What Changed

- Behavior differences:
  - interpreter-backed active pipelines now expose `PID` in admin `ps` output while running.
  - admin `kill <wid>` can terminate prompt-blocked interpreter tasks quickly.
  - RPC mechanism failures include richer error class context.
- Startup/config/default differences:
  - none.
- Identity/routing/connector differences:
  - none.

## Operator Actions Required

- Required config changes:
  - none.
- Optional config changes:
  - none.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  - deploy, validate `ps/kill` on an interactive interpreter prompt, then run full integration suites.
- Rollback/fallback instructions:
  - revert execution-model-slice8 commit(s) to restore prior behavior.
- Known temporary limitations:
  - cancellation remains process-oriented, not fine-grained method/task cancellation.

## Validation

- How to verify success:
  - `go test ./bot`
  - `TEST=GoFull make integration`
  - `TEST=JSFull make integration`
  - `TEST=LuaFull make integration`
- How to detect failure quickly:
  - `kill` reports no PID for active interpreter tasks, or prompt waits remain stuck after kill.

## References

- Impact report:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice8/impact-surface-report.md`
- PR checklist:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice8/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/multiprocess/ARCHITECTURE_DECISIONS.md`
