# PR Invariants Checklist

## Scope

- PR slice: `prompt-timeout-ssh-terminal-slice5`
- Linked impact report: `aidocs/multi-protocol/2026-02-13-prompt-timeout-ssh-terminal-slice5/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: yes
- Control flow remains explicit: yes
- Shared authorization/business policy logic remains in engine flows: yes
- Permission decisions use protocol-agnostic username: yes
- Per-connector message ordering guarantees preserved: yes
- Config precedence rules remain explicit: yes

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username: yes (unchanged)
- Cross-protocol identity mapping is explicit (no heuristic inference): yes (unchanged)
- Connector-local behavior does not bypass engine policy rules: yes
- Cross-connector isolation maintained (if multiple connectors enabled): yes
- Failure in one connector does not terminate others (if multiple enabled): yes

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`: yes
- Config default/override behavior validated: yes (no config surface change)
- Operator-visible changes documented: yes
- Compatibility note completed (or explicitly not required): yes

## Tests

- Focused tests added/updated: yes (`bot/replyprompt_test.go`)
- Existing relevant tests passing: yes (`go test ./bot`)
- Broader test pass status recorded: yes (`go test ./...`)

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved: n/a
- Connector docs updated where behavior changed: yes (`aidocs/SSH_CONNECTOR.md`)
- Other affected docs updated: yes (`aidocs/PIPELINE_LIFECYCLE.md`, `aidocs/EXTENSION_API.md`)

## Sign-Off

- Residual risks:
  - Prompt timeout behavior depends on task type and extension; if additional interpreter types are added later, timeout classification must be extended.
- Follow-up items:
  - None required for this slice.
