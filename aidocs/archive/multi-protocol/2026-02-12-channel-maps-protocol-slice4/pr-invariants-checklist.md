# PR Invariants Checklist

## Scope

- PR slice: `channel-maps-protocol-slice4`
- Linked impact report: `aidocs/multi-protocol/2026-02-12-channel-maps-protocol-slice4/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: `yes`
- Control flow remains explicit: `yes`
- Shared authorization/business policy logic remains in engine flows: `yes`
- Permission decisions use protocol-agnostic username: `yes` (unchanged in this slice)
- Per-connector message ordering guarantees preserved: `yes`
- Config precedence rules remain explicit: `yes`

## Multi-Protocol / Connector

- Transport-specific internal IDs map explicitly to shared username: `yes` (unchanged; existing per-protocol user map behavior preserved)
- Cross-protocol identity mapping is explicit (no heuristic inference): `yes`
- Connector-local behavior does not bypass engine policy rules: `yes`
- Cross-connector isolation maintained (if multiple connectors enabled): `yes`
- Failure in one connector does not terminate others (if multiple enabled): `yes`

## Startup / Config / Compatibility

- Startup and load order verified against `aidocs/STARTUP_FLOW.md`: `yes` (no startup ordering changes)
- Config default/override behavior validated: `yes` (no schema changes; map population only)
- Operator-visible changes documented: `yes` (compatibility note included)
- Compatibility note completed (or explicitly not required): `yes`

## Tests

- Focused tests added/updated: `yes` (`TestSendProtocolUserChannelMessageRouting`, `TestProtocolChannelLookupPrefersProtocolScopedMaps` in `bot/connector_runtime_test.go`)
- Existing relevant tests passing: `yes` (`go test ./bot`)
- Broader test pass status recorded: `yes` (`go test ./...`)

## Documentation

- `aidocs/COMPONENT_MAP.md` updated if component boundaries moved: `n/a` (no component boundary change)
- Connector docs updated where behavior changed: `n/a` (connector behavior unchanged; engine lookup path only)
- Other affected docs updated: `yes` (slice4 docs and slice3 checklist residual-risk cleanup)

## Sign-Off

- Residual risks:
  - legacy global channel map fallback remains by design for compatibility; duplicate channel names without protocol context can still resolve to global fallback in protocol-unknown paths.
- Follow-up items:
  - consider tightening fallback behavior in a future cleanup once all send/lookup call sites carry explicit protocol context.
