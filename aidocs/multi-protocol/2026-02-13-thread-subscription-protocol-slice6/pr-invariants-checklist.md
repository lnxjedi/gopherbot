# PR Invariants Checklist

## Scope

- PR slice: `thread-subscription-protocol-slice6`
- Linked impact report: `aidocs/multi-protocol/2026-02-13-thread-subscription-protocol-slice6/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: yes
- Control flow remains explicit: yes
- Shared authorization/business policy logic remains in engine flows: yes
- Permission decisions use protocol-agnostic username: yes
- Per-connector message ordering guarantees preserved: yes
- Config precedence rules remain explicit: yes

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username: yes
- Cross-protocol identity mapping is explicit (no heuristic inference): yes
- Connector-local behavior does not bypass engine policy rules: yes
- Cross-connector isolation maintained (if multiple connectors enabled): yes (improved for subscriptions)
- Failure in one connector does not terminate others (if multiple enabled): yes

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`: yes (unchanged)
- Config default/override behavior validated: yes
- Operator-visible changes documented: yes
- Compatibility note completed (or explicitly not required): yes

## Tests

- Focused tests added/updated: yes (`bot/subscribe_thread_test.go`)
- Existing relevant tests passing: yes (`go test ./bot`)
- Broader test pass status recorded: yes (`go test ./...`, `make test`)

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved: n/a
- Connector docs updated where behavior changed: n/a
- Other affected docs updated: yes (`aidocs/TESTING_CURRENT.md`)

## Sign-Off

- Residual risks:
  - restored legacy subscription keys without protocol still require compatibility lookup path.
- Follow-up items:
  - optional future cleanup: migrate persisted legacy subscription keys to protocol-scoped format on save.
