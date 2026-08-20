# Testing

## Quick commands

- `make test` runs unit tests and the local WireGuard check, then builds the
  process-backed integration runner.
- `make integration-run TEST=<selector>` runs one or more YAML integration
  suites; omit `TEST` to run all suites.
- `make integration-mcp TEST=<selector>` runs through the MCP wrapper used by
  AI-assisted validation.
- `make testbot` builds the interactive test bot used while developing fixture
  configurations.

## What runs where

- Unit tests are ordinary Go tests run by `go test ./...`.
- Integration suites are YAML files under `integration/suites/data/`. Each
  suite runs in its own robot process through `gopherbot-integration`.
- Robot configurations and extension fixtures live under `test/`.

List available suites with `./gopherbot-integration list-suites`. Selectors may
be an exact suite name, a glob, a comma-separated list, or metadata such as
`subsystem:`, `tag:`, `runtime:`, or `tier:`.

## Nudge: more unit tests welcome

Integration tests provide good coverage for end-to-end behavior, but we still have gaps
in unit tests around connector and routing logic. If you’re touching behavior that routes
or transforms messages, consider adding a small unit test to guard it.
