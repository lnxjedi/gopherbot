# Developing Integration Tests

Gopherbot's integration tests are YAML suites under
`integration/suites/data/`. Each suite starts a real robot process with the
`test` connector, sends messages as configured users and channels, and checks
the resulting replies and engine events.

Robot configurations and extension fixtures live under `test/`. A suite's
`config_dir` selects one of those configurations.

## Building and running suites

Build the runner and list the available suites:

```shell
make integration-build
./gopherbot-integration list-suites
```

Run one suite by name:

```shell
./gopherbot-integration run-suite TestPrompting
```

The selector may also be a glob, a comma-separated list, or metadata such as
`subsystem:`, `tag:`, `runtime:`, or `tier:`. To run every suite:

```shell
./gopherbot-integration run-suite all
```

`make integration-run TEST=<selector>` is a convenient wrapper. The MCP-backed
wrapper used by AI-assisted development is
`make integration-mcp TEST=<selector>`.

## Writing a suite

Start with a nearby YAML suite that uses the same fixture configuration or
runtime. A suite declares its name, metadata, `config_dir`, and cases. Each case
provides an input message and may assert replies and engine events.

For example:

```yaml
name: TestExample
metadata:
  subsystems:
    - routing
  tier: smoke
config_dir: test/example
cases:
  - input:
      user: alice
      channel: general
      text: ;ping
    replies:
      - user: alice
        channel: general
        text_pattern: PONG
    events:
      - CommandTaskRan
      - GoPluginRan
```

Keep expectations specific enough to prove the intended behavior without
depending on unrelated presentation details. Run the focused suite before a
broader selector.

## Failure artifacts and timeouts

Every run records a result, transcript, robot log, and runner log under
`integration/runs/<run-id>/`. Read the compact failure summary first, then use
the per-suite artifacts for diagnosis.

Cases have a bounded timeout. Suite startup and shutdown are also bounded; on a
hang, the runner saves a goroutine dump before terminating the suite process.
This isolation prevents one stuck robot from blocking the remaining suites.

## Interactive fixture development

`make testbot` builds an interactive robot with the test-oriented terminal
behavior. Run it from a fixture directory when developing configuration or
reproducing an exchange manually. The automated source of truth remains the
corresponding YAML suite.
