# PR Invariants Checklist

## Scope

- PR slice: `username-identity-contract-redesign` (implemented Slice 1 + Slice 2 + Slice 3 + Slice 3b + Slice 4 + Slice 5 + Slice 6 + Slice 7)
- Linked impact report: `aidocs/multi-protocol/2026-02-18-username-identity-contract-redesign/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: yes
- Control flow remains explicit: yes
- Shared authorization/business policy logic remains in engine flows: yes
- Permission decisions use protocol-agnostic username: yes
- Per-connector message ordering guarantees preserved: yes
- Config precedence rules remain explicit: yes

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username: yes (connector-local identity mapping)
- Cross-protocol identity mapping is explicit (no heuristic inference): yes
- Connector-local behavior does not bypass engine policy rules: yes
- Cross-connector isolation maintained (if multiple connectors enabled): yes
- Failure in one connector does not terminate others (if multiple enabled): yes
- Bot internal ID handling is protocol-scoped and deterministic by context: yes

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`: yes
- Config default/override behavior validated: yes
- Operator-visible changes documented: yes
- Compatibility note completed (or explicitly not required): yes

## Tests

- Focused tests added/updated: yes
- Existing relevant tests passing: yes
- Broader test pass status recorded: yes

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved: n/a
- Connector docs updated where behavior changed: yes (`aidocs/SSH_CONNECTOR.md`, `aidocs/SLACK_CONNECTOR.md`)
- Other affected docs updated: yes (`aidocs/STARTUP_FLOW.md`, `UPGRADING-v3.md`)

## Sign-Off

- Residual risks:
  - connectors must emit canonical usernames consistently for policy checks
  - legacy persisted ephemeral memories keyed by old `UserID` semantics may no longer be recalled
- Follow-up items:
  - none for this redesign package
