## Change Summary

- Change:
  - External Go (`.go` via yaegi) plugin/job/task execution now routes through `pipeline-child-rpc`.
  - External Go plugin default-config retrieval now routes through `go_get_config` over RPC.
- Why:
  - Complete migration of external interpreter-backed execution to the generic child RPC contract.
- Effective date/commit:
  - 2026-02-14 / execution-model-slice7 (working branch)

## What Stayed Compatible

- Unchanged behaviors:
  - compiled-in `taskGo`/`bot/*` handlers remain in-process.
  - connector routing, identity mapping, and authorization behavior remain unchanged.
  - Lua/JavaScript RPC behavior remains unchanged.
- Unchanged config/env surfaces:
  - no new required config keys or environment variables.

## What Changed

- Behavior differences:
  - external Go interpreted execution now runs out-of-process through child RPC.
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
  - deploy and validate with integration + Lua full; run JS full where environment allows local test HTTP listener.
- Rollback/fallback instructions:
  - revert execution-model-slice7 commit(s) to restore prior in-process Go execution.
- Known temporary limitations:
  - protocol cancellation/timeouts remain future work.

## Validation

- How to verify success:
  - `go test ./bot`
  - `make integration`
  - `TEST=LuaFull make test`
- How to detect failure quickly:
  - pipeline task failures reporting `go_*` RPC method errors in robot logs.

## References

- Impact report:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice7/impact-surface-report.md`
- PR checklist:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice7/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
  - `aidocs/multiprocess/ARCHITECTURE_DECISIONS.md`
