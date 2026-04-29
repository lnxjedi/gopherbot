# Integration Harness Plan

Status: implementation started, 2026-04-29.

This document describes the target integration-test architecture for replacing
the current `go test ./test` harness with a process-backed
`gopherbot-integration` workflow.

The goal is to keep integration tests close to manual QA: a real robot process
starts from normal configuration, receives messages from real configured users,
runs through the engine's routing/authorization/pipeline logic, executes
plugins and tasks, and exposes replies/events for comparison against expected
results.

## Impact Surface Report

### Change Summary

- Slice name: process-backed integration harness design.
- Goal: define a migration path from the current in-process Go test harness to
  a standalone `gopherbot-integration` command suitable for MCP automation,
  parallel suite execution, and real privilege-separation testing.
- Out of scope for this document: implementing the new command, moving tests,
  changing production startup behavior, or removing the existing `StartTest`
  harness.

### Subsystems Affected

Expected future code areas:

- `cmd/gopherbot-integration`: new integration command.
- `cmd/gopherbot-mcp`: lifecycle and suite-running tools for integration
  robots.
- `connectors/test`: test connector runtime, injection API, message/event
  capture, removal of `testing.T` coupling.
- `bot/aidev_http.go`: remains useful for manual/MCP interaction, but is not
  the primary suite-execution transport.
- `bot/start_t.go`: retained during migration, removed after coverage parity.
- `bot/emit_testing.go`: event capture remains test/integration-only and must
  become observable from an external runner.
- `Makefile`: build/run targets for `gopherbot-integration`.
- `test/`: existing suites, configs, helper types, and eventual suite registry.

Key symbols and current anchors:

- `Start` in `bot/start.go`
- `StartTest` in `bot/start_t.go`
- `initBot` and `run` in `bot/bot_process.go`
- `serveAIDevSendMessage`, `serveAIDevGetMessages`, and
  `serveAIDevSendAsRobot` in `bot/aidev_http.go`
- `TestConnector` in `connectors/test/connector.go`
- `testItem`, `testcases`, and `testcaseRepliesOnly` in `test/common_test.go`

### Current Behavior Anchors

Startup/order anchors:

- Production-like startup enters through `main.go` and `bot.Start`.
- Current integration tests enter through `StartTest`, not through process
  `main`.
- `StartTest` directly sets test config paths, initializes the connector
  runtime, starts `run`, waits for plugin-init quiescence, and returns the test
  connector to Go test code.

Routing/message-flow anchors:

- Current tests call `conn.SendBotMessage` directly.
- The test connector translates that into `robot.ConnectorMessage` and calls
  `IncomingMessage`.
- Replies are collected through an in-memory `speaking` channel.
- Events are collected through the `test` build-tag implementation of
  `GetEvents`.

Identity/authorization anchors:

- Test users and channels are defined by normal test robot config under
  `test/*/conf/protocols/test.yaml`.
- The engine still makes authorization decisions using canonical usernames.
- Hidden-message behavior is injected directly by test cases rather than
  inferred from slash syntax by the test connector.

Connector behavior anchors:

- The test connector is a normal registered connector named `test`.
- It currently depends on `*testing.T` for error/log reporting, which prevents
  use from a standalone process.
- It currently reports errors through `testing.T`, which prevents reuse from a
  standalone integration binary.

### Proposed Behavior

What changes:

- Add a separate `gopherbot-integration` command that shares the real engine and
  most production registration code, but includes integration-only code such as
  the test connector, suite runner, and event/query endpoints.
- Keep suite definitions in Go data structures and move them toward a reusable
  package that both the old Go test adapter and the new external runner can
  execute.
- Use the test connector as a scripted transport: it remains a normal connector
  goroutine, while the runner feeds scripted user messages and captures bot
  output for assertions.
- Make the test connector usable in a real robot process by removing
  `testing.T` coupling and adding injection/message-source APIs.
- Preserve event drain and plugin-init quiescence so existing event assertions
  can be preserved.

What does not change:

- Production `gopherbot` does not include the test connector or suite runner.
- Engine authorization, routing, elevation, and policy remain in engine flows.
- Connectors remain responsible for transport-local message shaping.
- The current `go test ./test` harness remains until the new process-backed
  harness reaches coverage parity.

### Invariant Impact Check

- Startup determinism preserved: yes. The new robot mode must enter through the
  same `bot.Start` path as production.
- Explicit control flow preserved: yes. Suite orchestration feeds a normal
  scripted connector and does not enter engine policy or pipeline internals.
- Shared auth/policy remains in engine flows: yes. The runner injects messages
  as configured users; it does not grant authorization.
- Permission checks remain username-based: yes. Test cases should use real
  configured users and groups from robot config.
- Connector ordering guarantees preserved: yes. Each test robot process owns
  its connector event loop and message buffers.
- Config precedence still explicit: yes. Suites continue to select config dirs
  such as `test/membrain` and `test/jsfull`.
- Multi-connector isolation preserved: yes. Integration-only connector APIs do
  not bypass runtime connector selection.

### Cross-Cutting Concerns

Startup sequencing impact:

- The integration robot process must not reintroduce an in-process `StartTest`
  shortcut for suites that are intended to validate startup or privsep behavior.
- Any suite-runner command mode in `gopherbot-integration` must intercept its
  own CLI commands before delegating normal robot startup to `bot.Start`.

Config loading impact:

- Existing test configs should remain normal robot config directories.
- Suite metadata should reference config directories rather than embedding
  alternate config behavior in the runner.

Execution ordering impact:

- Test steps must preserve the current sequence:
  wait for background plugin init, clear events, inject message, read expected
  replies, drain events, optional pause, wait/drain again.
- The external runner needs an API equivalent for "wait for quiescence" and
  "drain events" because it cannot call Go functions in the robot process.

Resource lifecycle impact:

- Every suite process should own a robot directory, log file, and result file.
- Parallel suites should run in separate robot directories or isolated copies of
  the same fixture directory.
- Shutdown should use a normal admin `quit` message first, then process
  termination as a fallback.

### Concurrency Risks

Shared state touched:

- Current in-process tests share package globals such as `currentCfg`,
  connector runtime state, event buffers, and brain state.
- The new harness should isolate suites by process to avoid shared global state.

Locking/channel/event-order assumptions:

- Test connector sends should remain normal connector sends. The runner may
  capture replies in linear order by default and can later correlate replies
  using inbound message IDs.
- Event drain APIs should keep the existing destructive drain semantics.

Race/deadlock/starvation risks:

- A suite runner that waits for exact messages without timeouts can hang.
- Parallel cases must not consume replies or events intended for another
  scripted exchange unless correlation has been explicitly implemented.
- Parallel suites must not share `test/workspace` or mutable brain directories.

Mitigations:

- Use per-suite temporary robot dirs.
- Use bounded waits with structured timeout failures.
- Keep full logs and machine-readable result files.
- Prefer suite-level process parallelism over sharing one robot process across
  unrelated suites.

### Backward Compatibility

Existing robots/config impact:

- None expected. This is an integration-only command and harness.

Behavior changes for operators/users:

- Eventually, `make integration` will no longer run `go test ./test` directly.
  It will build and/or run `gopherbot-integration`.
- Privsep integration tests will require explicit local setup because they need
  a setuid/setgid integration binary.

Migration/fallback plan:

- Keep the old harness during migration.
- Add a compatibility adapter so existing Go suite definitions can run in both
  the old and new runners.
- Remove `StartTest` only after coverage parity and explicit owner approval.

### Validation Plan

Focused tests during implementation:

- Unit tests for suite registry/filtering/result serialization.
- Unit tests for scripted test connector input/output behavior.
- Unit tests for suite registry/filtering/result serialization.

Broader regression tests:

- Existing `go test ./test` integration path until parity is proven.
- New `gopherbot-integration run-suite <name>` for each ported suite.
- Parallel suite execution against isolated robot dirs.

Manual verification:

- Run `gopherbot-integration run-suite TestBotName`.
- Confirm live interaction output and `result.json`.
- For privsep, install the binary setuid/setgid nobody and run only the
  privsep suite.

### Documentation Plan

- `aidocs/TESTING_CURRENT.md`: describe the current harness and point to this
  replacement plan.
- `aidocs/COMPONENT_MAP.md`: list this plan and, when implemented, the new
  command.
- `aidocs/STARTUP_FLOW.md`: update when `gopherbot-integration` exists and has
  defined CLI delegation behavior.
- `aidocs/EXECUTION_SECURITY_MODEL.md`: update when privsep-only integration
  suites exist.
- `UPGRADING-v3.md`: update only if operator-facing test/install workflows
  change.

## Target Architecture

The target has three layers:

1. A real integration robot process.
2. A runner that drives that process.
3. MCP tools that can build/start/run/fetch results without streaming large
   output into the model context.

### `gopherbot-integration`

`gopherbot-integration` should be a separate command under
`cmd/gopherbot-integration`.

It should share production engine code and the same startup path used by
`gopherbot`. The integration binary owns its top-level CLI, so no-argument
invocation and `--help` show integration-runner help. For high-fidelity robot
debugging, robot mode is explicit:

```text
gopherbot-integration run -aidev <token>
```

It should also expose integration-runner commands before delegating to
`bot.Start`, for example:

```text
gopherbot-integration list-suites
gopherbot-integration run-suite TestBotName
gopherbot-integration run-suite JSFull
```

The exact CLI can evolve, but the boundary should remain stable:

- explicit robot mode starts the engine;
- runner mode starts a scripted test-connector transport and asserts results;
- production `gopherbot` does not include runner code.

The integration command should include:

- the same built-in plugins/jobs/tasks needed by current tests;
- the `test` connector;
- the `test`/integration event capture implementation;
- suite registry and runner code;
- result/log file writers.

Production `modules.go` should remain production-owned. The integration command
can use an integration-specific registration file to import the same modules
plus the test connector.

### Test Connector

The test connector should become a process-safe, integration-capable connector.

Required changes:

- remove the `*testing.T` field and `ExportTest`;
- use `handler.Log` and returned errors instead of direct `testing.T` failures;
- keep user/channel validation against `ProtocolConfig`;
- keep the connector's normal `Run(stop)` goroutine and inbound message
  channel;
- assign scripted inbound messages stable `MessageID` values;
- capture outbound bot messages for assertions through the normal
  `SendProtocol*` methods;
- tag captured outbound messages with the originating inbound `MessageID` when
  available;
- preserve direct hidden injection through a boolean field;
- preserve `BasicMarkdown` rendering into readable text for assertions.

The runner should drive the connector as a scripted transport. This keeps the
test connector closer to terminal/SSH than to an HTTP polling API: user messages
arrive through the connector loop, and bot replies leave through connector send
methods.

### Correlation And Quiescence

Existing tests depend on two process-local helpers:

- `WaitForBackgroundInitsForTesting`
- `GetEvents`

The new runner should preserve equivalent behavior. The default runner remains
linear: wait for idle, clear events, send one scripted message, collect expected
replies, drain events, then advance.

For future parallel execution, the strongest correlation point is the incoming
connector message. The test connector can assign every scripted input a stable
`MessageID`. Worker and Robot send paths already pass `Incoming` back to
connector send methods, so the test connector can tag captured replies with the
originating `MessageID` without exposing worker internals to the harness.

Event assertions should remain linear at first because `emit(Event)` is global
today. Worker-scoped event correlation can be designed later if parallel cases
need event assertions.

### Suite Registry

The current suites are already close to a portable format because most tests
are Go data structures:

```go
[]testItem{
    {aliceID, general, ";ping", false,
        []TestMessage{{alice, general, "PONG", false}},
        []Event{CommandTaskRan, GoPluginRan},
        0,
    },
}
```

The migration should preserve that shape and move it into a reusable package.
A target structure could look like:

```go
type Suite struct {
    Name         string
    ConfigDir    string
    LogName      string
    FullGate     string
    Capabilities map[string]robot.ConnectorCapabilities
    Cases        []Case
    Flow         func(context.Context, Driver) error
}

type Case struct {
    Name        string
    Input       Message
    Replies     []ExpectedMessage
    Events      []bot.Event
    RepliesOnly bool
    Pause       time.Duration
}
```

The important part is not the exact names. The important part is that the old
Go test adapter and the new process-backed runner execute the same suite
definitions.

### Automated Porting Strategy

The porting work should be designed so a medium-reasoning coding pass can move
the bulk of tests mechanically.

The recommended path:

1. Create a suite package under `integration/`, for example
   `integration/suites`.
2. Move shared constants (`aliceID`, `general`, etc.) and message/case types
   into that package.
3. Move one simple suite, such as `TestBotName`, into a registered
   `Suite`.
4. Add a Go test compatibility adapter that executes the registered suite using
   the old in-process driver.
5. Add a process-backed driver that executes the same registered suite through
   the scripted test connector in `gopherbot-integration`.
6. Port ordinary `testcases(...)` suites by moving their `[]testItem` literals
   into registry entries.
7. Port `testcaseRepliesOnly(...)` flows by setting `RepliesOnly` on cases or
   flow steps.
8. Port dynamic tests using Go flow functions at first.

Dynamic tests include cases such as:

- extracting a validation code from one bot reply and using it in the next
  message;
- extracting a pipeline ID from `ps` output and using it in
  `get-pipeline-log`;
- waiting for operator alerts from timeout/failure workflows.

Those should remain Go flows initially. If they become repetitive, the runner
can later grow declarative capture/substitution fields, but that should not
block the migration.

### Driver Interface

The compatibility adapter and external runner should share a small driver
interface, for example:

```go
type Driver interface {
    WaitForIdle(ctx context.Context) error
    DrainEvents(ctx context.Context) ([]bot.Event, error)
    Send(ctx context.Context, msg Message) error
    Receive(ctx context.Context, want ExpectedMessage) (Message, error)
}
```

Two drivers can then exist:

- `InProcessDriver`: wraps the current `StartTest`, `SendBotMessage`,
  `GetBotMessage`, `GetEvents`, and `WaitForBackgroundInitsForTesting`.
- `ScriptedConnectorDriver`: runs inside a real `gopherbot-integration`
  process and drives the initialized test connector as a scripted transport.

This is the key to automated porting: suite data moves once, while drivers
decide whether execution is old-style or process-backed.

### Result Files

The runner should always write full results to files.

Suggested outputs:

- `result.json`: machine-readable suite/case/step status, timings, failure
  details, log paths, robot dir, binary path, PID, and privsep mode.
- `robot.log`: raw robot log.
- `transcript.txt`: scripted user/bot interaction transcript using the same
  `->` / `<-` lines shown by live output.
- `runner.log`: runner-level diagnostics.
- optional `messages.jsonl`: observed message stream.
- optional `events.jsonl`: observed event stream.

The CLI and MCP should return compact summaries plus file paths. They should
not stream full logs into model context by default.

For multi-suite invocations, the runner should allocate one timestamped
artifact directory for the whole invocation and place per-suite directories
under it. The final CLI summary should report that single directory as
`Results recorded in: <path>`, and MCP summaries should expose the same common
directory as `results_root`. Suite selection should support exact suite names,
`all`, simple glob patterns such as `TestShFull*`, and comma-separated selector
lists for MCP calls.

### Parallel Execution

Parallelism should be process-level.

Each suite worker should get:

- an isolated robot directory;
- an isolated workspace directory;
- unique robot and runner logs;
- independent brain data unless the suite explicitly tests persistence.

Avoid running unrelated suites inside the same robot process. The engine has
global runtime state, and process isolation gives simpler failure analysis.

## MCP Contract

The MCP should be designed into the harness from the start.

Existing `gopherbot-mcp` already supports starting arbitrary binaries with
`gopherbot_binary`, sending messages, polling messages, reading logs, and
stopping robots. That remains useful for manual robot interaction. Integration
suite execution should be suite-level: MCP runs `gopherbot-integration
run-suite ...` and reads the artifacts it writes.

`gopherbot-mcp` now shells out to `gopherbot-integration` for suite
orchestration. The MCP layer remains a thin runner: it does not embed test
expectations, does not inject individual scripted messages itself, and does not
inspect engine worker internals.

Current integration tools:

- `list_integration_suites`
- `run_integration_suite`
- `read_integration_result`

The behavior is:

- build or locate `gopherbot-integration`;
- let the integration command allocate isolated robot dirs;
- run selected suite/case filters;
- write full artifacts to disk;
- return a compact result summary.

MCP should not embed test expectations. Expectations belong in the Go suite
registry used by both local CLI and MCP-driven runs.

Manual robot lifecycle tools (`start_robot`, `send_message`, `get_messages`,
and `stop_robot`) remain available for exploratory QA, but full suite execution
should use `run_integration_suite` so logs/results are file-backed and
repeatable.

## Privsep Integration Tests

Privsep tests require a real executable with specific ownership and mode bits.
This is the strongest reason to move away from `go test ./test`.

The privsep suite should be separate from normal integration suites because it
requires local operator setup.

Candidate suite checks:

- startup rejects unmanaged supplementary groups by default;
- startup accepts explicitly allowed supplementary groups;
- unprivileged file-backed extension reports nobody UID/GID;
- privileged file-backed extension reports invoking robot UID/GID;
- unprivileged extension cannot read an invoking-user-owned `0600` file;
- privileged extension can read the same file;
- compiled-in Go extension remains trusted in-process and is not represented as
  an unprivileged execution mode;
- external Python/Ruby/Bash and built-in Lua/JS/Gsh/Yaegi paths all run in the
  expected child role.

Linux-only optional checks:

- UID-scoped firewall rules block nobody from EC2 IMDS endpoints;
- invoking robot user can still perform expected privileged host operations if
  explicitly configured.

The runner should detect whether the integration binary is installed for
privsep and skip/fail with a clear setup message depending on the requested
suite mode.

## Make Targets

During migration:

```text
make integration
make integration-build
make integration-run TEST=BotName
make integration-legacy
```

During this phase, `make integration` should build `gopherbot-integration` and
print concise instructions for listing and running suites. `make
integration-legacy` should keep the old `go test ./test` workflow available
until parity is proven.

After parity:

```text
make integration
```

should continue to build `gopherbot-integration` and print the suite-selection
instructions unless the owner explicitly changes the default to run a suite set.

Privsep should be explicit:

```text
make integration-privsep-check
make integration-privsep TEST=PrivsepExternalPython
```

Those targets should verify binary ownership/mode before running and should not
silently attempt privileged setup.

## Migration Phases

### Phase 1: Compatibility Spine

- Add `cmd/gopherbot-integration`.
- Add integration-specific module imports.
- Extract suite types/constants to a non-`_test` package.
- Keep old Go tests running through an adapter.
- Remove `testing.T` from the test connector.

Exit criteria:

- One simple suite runs both through old `go test` and through the new process
  runner using the same suite definition.

### Phase 2: Connector And Runner Observability

- Preserve the test connector's normal goroutine/channel behavior while making
  it usable without `testing.T`.
- Add inbound `MessageID` assignment and reply correlation fields.
- Add result/log file writing.

Exit criteria:

- The process-backed runner can execute one suite without direct in-process
  function calls.

### Phase 3: Mechanical Suite Port

- Port ordinary `testcases` suites by moving literals into the suite registry.
- Port replies-only flows with a `RepliesOnly` flag.
- Keep dynamic tests as Go flow functions.

Exit criteria:

- A medium-reasoning coding pass can continue porting test files using the
  established pattern.

### Phase 4: MCP Integration

- [x] Add MCP tools or wrappers for integration suite execution.
- [x] Keep full logs/results in files and return summaries.
- [ ] Support case filters.

Exit criteria:

- Codex can run focused integration suites without ad-hoc shell prompts or
  streaming full output into context.

### Phase 5: Privsep Suites

- Add host-setup detection for setuid/setgid `gopherbot-integration`.
- Add privsep-only suites.
- Document setup and cleanup.

Exit criteria:

- Privsep behavior is tested through a real process boundary rather than
  inferred from unit tests or manual logs.

### Phase 6: Retirement

- Confirm parity with the old `go test ./test` harness.
- Remove or archive `StartTest`.
- Remove the old direct connector harness from `test/common_test.go`.
- Update `make integration` to use only the new harness.

Exit criteria:

- `go test` is for package/unit tests.
- `gopherbot-integration` is the only integration-test runner.

## Open Design Questions

- Should event correlation become worker/message-scoped later, or is
  suite-level process parallelism enough?
- Should `make integration` eventually run the default suite set, or only build
  and print instructions?
- Should case filters live in suite definitions, CLI flags, or both?

## Recommended First Vertical Slice

The first coding slice should be intentionally narrow:

1. Add `cmd/gopherbot-integration` that can run the engine like `gopherbot`.
2. Include the test connector and current test module imports.
3. Remove `testing.T` from the test connector without changing existing test
   behavior.
4. Define suite/driver types.
5. Move `TestBotName` into a registered suite.
6. Keep the old `go test` path executing that suite through an in-process
   driver.
7. Add enough scripted test-connector support to execute the same suite through
   a real `gopherbot-integration` process.

After that vertical slice is green, most suite porting should be mechanical.
