# Testing Operations

This file retains operational instructions that are not safe to infer from test
source alone.

## Default workflow

- Run focused package tests first, then `go test ./...` or `make unit` when the
  scope warrants it.
- `make test` runs unit and local script checks but only builds the
  process-backed integration runner; it does not execute all integration
  suites.
- `lib/` is a separate Go module. Run its tests from `lib/` with
  `GOWORK=off`; do not force root-module globs across module boundaries.
- For core engine or connector-runtime changes, finish with `make`.

## Process-backed integration

The authoritative suites are YAML under `integration/suites/data/` and run in a
real robot process. The Go tests under `test/` are a legacy compatibility
harness.

For AI validation:

1. Build `gopherbot-mcp` and the runner with
   `make mcp integration-build` when needed.
2. Use the MCP `list_integration_suites` tool to discover suites/selectors.
3. Use `run_integration_suite` with an exact suite, glob, comma-separated
   selectors, or metadata selector (`subsystem:`, `tag:`, `runtime:`, `tier:`)
   and live output disabled.
4. Read the compact result first. Inspect the saved `result.json`,
   `runner.log`, `robot.log`, transcript, or timeout goroutine dump only when
   needed.

`make integration-mcp TEST=<selector>` is the local wrapper. Do not use direct
`go test ./test`, `make integration-legacy`, or `make integration-full` unless
the owner explicitly requests the legacy harness.

Each suite has a case timeout (normally 14 seconds). A timeout hard-exits the
suite after saving artifacts so the goroutine dump reflects the hang. A
shutdown timeout is recorded separately.

## Failure discipline

Before changing code or expectations, classify every integration failure:

- regression/new bug; or
- intentional behavior change with outdated expectation.

Do not update assertions merely to make a suite pass. For 1–3 changed tests,
report the exact issue each expectation fixes; group larger migrations.

A localhost-listener `operation not permitted` error is commonly an execution
sandbox restriction, not product behavior. Verify environment permissions
before classifying it as a regression.

## Extension development

Use the built binary rather than standalone language runtimes:

- `./gopherbot syntax [-json] <file...>` for `.lua`, `.js`, `.gsh`, and
  interpreted `.go`.
- `./gopherbot script [-fixture <yaml|json>] <file> -- <command> [args...]`
  for a child-RPC run with a fixture-backed Robot API.
- `./gopherbot script -new-fixture <path>` copies the documented default
  fixture.

These checks intentionally do not start connectors, queues, modules, the real
brain, or the HTTP listener. They do not replace integration coverage for
routing, authorization, identity, scheduling, or persistence.

`test-scripts/` contains examples. `make wireguard-plugin-test` is the
repository-specific dry-run check and is included in normal test targets.

## Privsep

Normal suites cannot prove setuid behavior. Use the manual host procedure in
`AGENTS.md` for privsep changes. Keep setuid-only suites separate from ordinary
CI because they require controlled executable ownership and mode.
