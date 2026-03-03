# PR Invariants Checklist (Draft)

Use this checklist before merge. Mark each item `yes`, `no`, or `n/a`.

## Scope

- PR slice: `config-identity-slice1`
- Linked impact report: `aidocs/multi-protocol/2026-02-11-config-identity-slice1/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: pending
- Control flow remains explicit: pending
- Shared authorization/business policy logic remains in engine flows: pending
- Permission decisions use protocol-agnostic username: pending
- Per-connector message ordering guarantees preserved: pending
- Config precedence rules remain explicit: pending

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username: pending
- Cross-protocol identity mapping is explicit (no heuristic inference): pending
- Connector-local behavior does not bypass engine policy rules: pending
- Cross-connector isolation maintained (if multiple connectors enabled): n/a in this slice
- Failure in one connector does not terminate others (if multiple enabled): n/a in this slice

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`: pending
- Config default/override behavior validated: pending
- Operator-visible changes documented: pending
- Compatibility note completed (or explicitly not required): pending

## Tests

- Focused tests added/updated: pending
- Existing relevant tests passing: pending
- Broader test pass status recorded: pending

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved: pending
- Connector docs updated where behavior changed: n/a in this slice
- Other affected docs updated: pending

## Sign-Off

- Residual risks: duplicate username attribute collisions across protocols remain unresolved for later slice design if shared profile fields diverge.
- Follow-up items: multi-connector runtime orchestration, connector lifecycle admin controls, cross-protocol send API.
