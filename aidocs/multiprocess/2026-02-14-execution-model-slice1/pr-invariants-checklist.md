# PR Invariants Checklist

## Scope

- PR slice: `execution-model-slice1`
- Linked impact report: `aidocs/multiprocess/2026-02-14-execution-model-slice1/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: `yes`
- Control flow remains explicit: `yes` (task execution now has explicit `executeTask` boundary)
- Shared authorization/business policy logic remains in engine flows: `yes`
- Permission decisions use protocol-agnostic username: `yes` (unchanged)
- Per-connector message ordering guarantees preserved: `yes` (unchanged)
- Config precedence rules remain explicit: `yes` (unchanged)

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username: `yes` (unchanged)
- Cross-protocol identity mapping is explicit (no heuristic inference): `yes` (unchanged)
- Connector-local behavior does not bypass engine policy rules: `yes` (unchanged)
- Cross-connector isolation maintained (if multiple connectors enabled): `yes` (unchanged)
- Failure in one connector does not terminate others (if multiple enabled): `yes` (unchanged)

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`: `yes` (no startup flow changes)
- Config default/override behavior validated: `yes` (no config changes)
- Operator-visible changes documented: `yes` (compatibility note)
- Compatibility note completed (or explicitly not required): `yes`

## Tests

- Focused tests added/updated: `yes` (`bot/task_execution_test.go`)
- Existing relevant tests passing: `yes` (`go test ./bot`)
- Broader test pass status recorded: `yes` (`make integration`)

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved: `yes`
- Connector docs updated where behavior changed: `n/a` (no connector behavior changes)
- Other affected docs updated: `yes` (`aidocs/EXECUTION_SECURITY_MODEL.md`, `aidocs/PIPELINE_LIFECYCLE.md`)

## Sign-Off

- Residual risks:
  - This slice introduces only an abstraction seam; process execution behavior is not implemented yet.
- Follow-up items:
  - Add process runner implementation for non-Go tasks behind `executeTask`.
  - Define parent/child IPC protocol and lifecycle management in next slice.
