# PR Invariants Checklist

## Scope

- PR slice: `protocol-send-user-channel-slice3`
- Linked impact report: `aidocs/multi-protocol/2026-02-12-protocol-send-user-channel-slice3/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: `yes`
- Control flow remains explicit: `yes`
- Shared authorization/business policy logic remains in engine flows: `yes`
- Permission decisions use protocol-agnostic username: `yes`
- Per-connector message ordering guarantees preserved: `yes`
- Config precedence rules remain explicit: `yes`

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username: `yes` (new API resolves user IDs via protocol roster map first)
- Cross-protocol identity mapping is explicit (no heuristic inference): `yes`
- Connector-local behavior does not bypass engine policy rules: `yes`
- Cross-connector isolation maintained (if multiple connectors enabled): `yes`
- Failure in one connector does not terminate others (if multiple enabled): `yes`

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`: `yes` (no startup flow changes in this slice)
- Config default/override behavior validated: `yes` (no config behavior changes)
- Operator-visible changes documented: `yes`
- Compatibility note completed (or explicitly not required): `yes`

## Tests

- Focused tests added/updated: `yes` (`TestSendProtocolUserChannelMessageRouting` in `bot/connector_runtime_test.go`)
- Existing relevant tests passing: `yes` (`go test ./bot ./modules/javascript ./modules/lua ./modules/yaegi-dynamic-go ./connectors/test ./connectors/slack ./connectors/ssh`)
- Broader test pass status recorded: `yes` (`go test ./...`)

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved: `n/a` (no component boundary changes)
- Connector docs updated where behavior changed: `n/a` (connector semantics unchanged; API is engine-level)
- Other affected docs updated: `yes` (`aidocs/EXTENSION_API.md`, JS/Lua method checklists, slice docs)

## Sign-Off

- Residual risks:
  - channel resolution remains global map-based; protocol-specific channel ambiguity is unchanged from current behavior.
  - unknown protocol targeting returns `Failed` (not a more specific return code).
- Follow-up items:
  - consider protocol-scoped channel lookup once per-protocol channel identity collisions become a practical issue.
