# PR Invariants Checklist Template

Use this checklist before merge. Mark each item `yes`, `no`, or `n/a`.

## Scope

- PR slice:
- Linked impact report:

## Core Invariants

- Startup sequence remains deterministic and traceable:
- Control flow remains explicit:
- Shared authorization/business policy logic remains in engine flows:
- Permission decisions use protocol-agnostic username:
- Per-connector message ordering guarantees preserved:
- Config precedence rules remain explicit:

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username:
- Cross-protocol identity mapping is explicit (no heuristic inference):
- Connector-local behavior does not bypass engine policy rules:
- Cross-connector isolation maintained (if multiple connectors enabled):
- Failure in one connector does not terminate others (if multiple enabled):

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`:
- Config default/override behavior validated:
- Operator-visible changes documented:
- Compatibility note completed (or explicitly not required):

## Tests

- Focused tests added/updated:
- Existing relevant tests passing:
- Broader test pass status recorded:

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved:
- Connector docs updated where behavior changed:
- Other affected docs updated:

## Sign-Off

- Residual risks:
- Follow-up items:

