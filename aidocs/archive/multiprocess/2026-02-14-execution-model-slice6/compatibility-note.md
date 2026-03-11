## Change Summary

- Change:
  - External JavaScript task/plugin/job execution now routes through `pipeline-child-rpc`.
  - JavaScript plugin default-config retrieval now routes through `js_get_config` over RPC.
- Why:
  - Extend the generic interpreter RPC contract to JavaScript as the second migrated interpreter.
- Effective date/commit:
  - 2026-02-14 / execution-model-slice6 (working branch)

## What Stayed Compatible

- `taskGo`/`bot/*` compiled-in handlers remain in-process.
- External executable non-interpreter path remains unchanged.
- External Go interpreter path remains in-process in this slice.
- Connector, identity, authorization, and routing semantics are unchanged.

## What Changed

- Runtime behavior:
  - JavaScript interpreter-backed execution now runs out-of-process via `pipeline-child-rpc`.
- Internal protocol behavior:
  - RPC method surface now includes `js_run` and `js_get_config`.

## Operator Actions Required

- Required config/env changes:
  - none
- Optional changes:
  - none

## Rollout / Fallback

- Recommended rollout:
  - deploy and validate with JS full integration suite.
- Rollback:
  - revert slice 6 commit(s) to return JS execution to in-process path.

## Validation

- `go test ./bot`
- `make integration`
- `RUN_FULL=js make test`

## References

- Impact report:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice6/impact-surface-report.md`
- PR checklist:
  - `aidocs/multiprocess/2026-02-14-execution-model-slice6/pr-invariants-checklist.md`
