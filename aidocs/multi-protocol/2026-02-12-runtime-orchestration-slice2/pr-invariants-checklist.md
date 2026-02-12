# PR Invariants Checklist

## Scope

- PR slice: `runtime-orchestration-slice2`
- Linked impact report: `aidocs/multi-protocol/2026-02-12-runtime-orchestration-slice2/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: `yes`
- Control flow remains explicit: `yes`
- Shared authorization/business policy logic remains in engine flows: `yes`
- Permission decisions use protocol-agnostic username: `yes` (existing username checks unchanged)
- Per-connector message ordering guarantees preserved: `yes` (ordering remains connector-local)
- Config precedence rules remain explicit: `yes`

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username: `yes` (via per-protocol roster loading + merged user roster)
- Cross-protocol identity mapping is explicit (no heuristic inference): `yes` (roster-driven)
- Connector-local behavior does not bypass engine policy rules: `yes`
- Cross-connector isolation maintained (if multiple connectors enabled): `yes`
- Failure in one connector does not terminate others (if multiple enabled): `yes` (secondary failures isolated)

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`: `yes` (updated)
- Config default/override behavior validated: `yes`
- Operator-visible changes documented: `yes` (compatibility note + startup docs)
- Compatibility note completed (or explicitly not required): `yes`

## Tests

- Focused tests added/updated: `yes` (existing config helper tests retained; runtime path validated with package tests)
- Existing relevant tests passing: `yes` (`go test ./bot`)
- Broader test pass status recorded: `yes` (`go test ./...`)

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved: `yes`
- Connector docs updated where behavior changed: `n/a` (connector internals unchanged in this slice)
- Other affected docs updated: `yes` (`aidocs/STARTUP_FLOW.md`, `aidocs/TESTING_CURRENT.md`)

## Sign-Off

- Residual risks:
  - Slack connector package-global `started` guard may reject re-initialize after stop/restart in-process.
  - If two protocols share identical raw internal IDs, merged `UserRoster` ID lookup remains ambiguous until protocol-scoped ID maps land.
- Follow-up items:
  - Add focused runtime-manager unit tests with fake connectors.
  - Resolve connector restartability constraints where package-global state exists.
