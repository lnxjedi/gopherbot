## Change Summary

- Change:
  - External Lua task/plugin/job execution now routes through `pipeline-child-rpc`.
  - Lua plugin default-config retrieval now routes through `lua_get_config` over RPC.
- Why:
  - Activate process isolation for the first interpreter on the generic RPC contract.
- Effective date/commit:
  - 2026-02-14 / execution-model-slice5 (working branch)

## What Stayed Compatible

- `taskGo`/`bot/*` compiled-in handlers remain in-process.
- External executable non-interpreter child-runner behavior is unchanged.
- External `.go` and `.js` interpreter execution remains in-process in this slice.
- No connector/identity/policy behavior changes.

## What Changed

- Runtime behavior:
  - Lua interpreter-backed calls now execute out-of-process via `pipeline-child-rpc`.
- Internal protocol behavior:
  - RPC method surface now includes `lua_run`, `lua_get_config`, and parent-served `robot_call`.

## Operator Actions Required

- Required config/env changes:
  - none
- Optional changes:
  - none

## Rollout / Fallback

- Recommended rollout:
  - deploy and run Lua full integration suite.
- Rollback:
  - revert slice 5 commit(s) to return Lua execution to in-process path.

## Validation

- `go test ./bot`
- `make integration`
- `RUN_FULL=lua make test`

## References

- Impact report:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice5/impact-surface-report.md`
- PR checklist:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice5/pr-invariants-checklist.md`
