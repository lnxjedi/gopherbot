# Testing (Current)

Focus: integration test harness under `test/` and how tests are executed.

## How tests are discovered and run

- Integration tests are gated by build tags like `//go:build integration` in files such as `test/common_test.go` and `test/bot_integration_test.go`.
- `make test` runs `go test ... ./test` with tags `test integration netgo osusergo static_build` and optional `TEST` filter (see `Makefile` target `test`).
- The test harness depends on test-only event emission: `bot/emit_testing.go` is built with `//go:build test`, while `bot/emit_noop.go` is used when the `test` tag is not set.

## Harness entrypoints

- `setup()` in `test/common_test.go` starts the bot via `StartTest()` and relies on the test configs to choose the `test` protocol.
- `setupWithOptions()` in `test/common_test.go` is the per-suite harness entrypoint when a test block needs connector capability overrides (for example simulating a protocol without hidden-command support while still using the `test` connector).
- `StartTest()` is defined in `bot/start_t.go` (only built with `test` tag). It locates the repo root by walking up until `conf/robot.yaml` is found, `chdir`s there so startup mode resolves to `test-dev`, initializes connector runtime via `initializeConnectorRuntime`, runs `run()`, then waits for the current async plugin-init batch to drain before clearing startup events and returning control to the harness (see also `bot/bot_process.go`, `bot/tasks.go`, and `bot/connector_runtime.go`).
- The test connector is registered in `connectors/test/init.go` with `robot.RegisterConnector("test", Initialize)`, and `Initialize(...)` returns `robot.InitializedConnector{Connector, Capabilities}` so the harness can treat hidden-command support as a runtime capability rather than a registration-time constant.
- The test connector derives bot display/full name from `BotInfo` in `conf/robot.yaml` via `Handler.GetBotInfo()`, not from protocol-local `BotName`/`BotFullName` fields.
- The test connector runtime loop lives in `connectors/test/connector.go` (`(*TestConnector).Run`).
- For `BasicMarkdown` bot output, the test connector stores a plain-text rendering rather than raw markdown markers so integration assertions can match readable user-visible content; raw authored markdown remains covered by targeted unit tests in `bot/` and `robot/util/`.

## setup / teardown / testcases flow

- `setup(cfgdir, logfile, t)` sets `GOPHER_ENCRYPTION_KEY`, wires `testc.ExportTest.Test`, and calls `StartTest()` (see `test/common_test.go`).
- `setupWithOptions(cfgdir, logfile, opts, t)` does the same, but first applies test-only connector capability overrides through `ApplyConnectorCapabilitiesForTesting(...)` in `bot/connector_capabilities_testing.go`, then restores them during teardown/cleanup.
- `teardown(t, done, conn)` sends a `quit` message and waits on `done`, then checks emitted events from `GetEvents()` (see `test/common_test.go` and `bot/emit_testing.go`).
- `teardownWithOptions(t, done, conn, cleanup)` is the paired helper for suites that used `setupWithOptions(...)`.
- `testcases(t, conn, tests)` drives each test by:
  - Waiting for any in-flight async plugin-init batch to finish, then clearing startup/background-init events with `GetEvents()` before sending the message (see `test/common_test.go`).
  - Sending the message through `conn.SendBotMessage` (type `testc.TestMessage` in `connectors/test/connector.go`).
  - Matching expected replies using regex on `TestMessage.Message` and strict equality on user/channel/threaded fields (see `test/common_test.go`).
  - Comparing observed `[]Event` from `GetEvents()` with expected events (event type `Event` in `bot/events.go`), then waiting/draining again so late init events from reload-like flows do not leak into later cases or teardown.

## Integration Failure Triage

When `make test` fails in the `integration` target, check these first:

1. The failing test name and first fatal line in `go test -v` output.
2. The bot runtime log file used by the harness:
   - usually `/tmp/bottest.log`
   - builtins suite uses `/tmp/bottest-builtins.log`
3. The config path printed by `StartTest()` (for example `test/membrain`, `test/jsfull`, `test/luafull`, `test/shfull`, `test/gofull`).

Log file paths are passed explicitly by integration tests through `setup(..., logfile, ...)` in `test/common_test.go`.

Required triage rule:

- Every integration failure must be classified before changing code or test expectations:
  - real regression / newly introduced bug
  - intentional behavior change with outdated test expectations
- Do not update assertions blindly just to make `make test` pass.
- The expected end state for a task is that applicable integration tests run cleanly after this classification work.

Common symptom:

- `Fatal: Listening on tcp4 port 127.0.0.1:0 ... operation not permitted`
  - means the test process could not open a localhost listener in the current execution environment (sandbox/permissions issue), not necessarily a bot logic regression.

## Live Brain Inspection (Manual Debugging)

For robot-side data shape debugging (for example long-term memory/datum format mismatches), use the robot CLI from a custom robot directory:

- `../gopherbot/gopherbot list`
- `../gopherbot/gopherbot fetch <full_key>`

Example (`openai-fallback` conversation records):

- `../gopherbot/gopherbot list | rg "openaifallback:conversation|conversation:index"`
- `../gopherbot/gopherbot fetch openai-fallback:openaifallback:conversation:index:v1`
- `../gopherbot/gopherbot fetch openai-fallback:openaifallback:conversation:v2:<sha1>`

Notes:

- `list` returns fully namespaced datum keys (for plugins this is typically `<plugin-name>:<datum-key>`).
- `fetch` returns raw JSON payloads, which is useful for confirming actual stored shape versus expected struct shape.

## Test case structure

- `testItem` in `test/common_test.go` defines a case as:
  - `user`, `channel`, `message`, `threaded` (input fields).
  - `replies []TestMessage` where `TestMessage.Message` is a regex to match output (type `TestMessage` in `test/common_test.go`).
  - `events []Event` expected for the interaction (type `Event` in `bot/events.go`).
  - `pause` (milliseconds) to sleep between cases (see `test/common_test.go`).
- Hidden messages are signaled by a leading `/` in `message` and transformed before sending (see `test/common_test.go`).
- The harness sets the test connector's `Hidden` flag directly; the test connector does not parse slash syntax itself. That is intentional so integration suites can simulate protocols with and without hidden-command support using the same connector.
- Hidden admin/history/job-command coverage should continue using that same mechanism rather than trying to emulate connector-local slash parsing in the test connector.

## Protocol selection for tests

- The integration configs under `test/*/conf/robot.yaml` set `PrimaryProtocol: {{ env "GOPHER_PROTOCOL" | default "test" }}` so tests use the test connector by default without setting env vars in Go code.
- When running a config interactively (e.g., `cd test/membrain` + `gopherbot`), set `GOPHER_PROTOCOL=terminal` (via `private/environment`) to exercise the terminal connector.

## `make testbot` (interactive test helper)

- `make testbot` builds `gopherbot` with the `test` build tag, enabling test-only behavior like event capture (`bot/emit_testing.go`) and terminal send formatting (`bot/term_sendmessage_tbot.go`).
- In the terminal connector, pressing `<enter>` with no input prints buffered events from `GetEventStrings()` (see `bot/term_connector.go` and `bot/emit_testing.go`).

## Representative test suites

- Core bot behavior and message matching: `test/bot_integration_test.go` (e.g., `TestBotName`, `TestMessageMatch`).
- Memory tests: `test/memory_integration_test.go`.
- Lists plugin behavior: `test/lists_integration_test.go`.
- External Python/Ruby EncryptSecret coverage: `test/external_encrypt_integration_test.go`.
- External yaegi Go full coverage: `test/go_full_test.go`.
- Gopherbot shell full coverage: `test/sh_full_test.go` plus `plugins/test/shfull.gsh`.
- JavaScript full coverage: `test/js_full_test.go` plus `plugins/test/jsfull.js`, including the OAuth2 link/get/unlink engine API cycle.
- Admin/watchdog coverage: `test/admin_watchdog_integration_test.go`, using `test/membrain/plugins/admininspect.sh` and `test/membrain/plugins/admintimeout.sh` to exercise:
  - hidden `ps` and `get-pipeline-log`
  - timeout warn/kill operator alerts for external pipelines
  - operator-facing failure alerts with stderr/traceback excerpts

## Focused routing/help metadata checks

- `go test ./bot ./modules/yaegi-dynamic-go`
- This focused set currently covers:
  - catch-all mode routing and selection in `bot/help_metadata_api_test.go`
  - engine-side help metadata filtering, ranking inputs, and deterministic fallback advice in `bot/help_metadata_api_test.go`
  - Yaegi symbol/runtime coverage for active Robot API methods in `modules/yaegi-dynamic-go/yaegi_dynamic_test.go`
  - shared `.yaegi-gopath` import coverage for installed (`gopherbot.internal/lib/...`) and custom (`robot.internal/lib/...`) interpreted-Go libraries

## Focused Admin/Watchdog Verification

- Bot unit package:
  - `go test ./bot`
  - includes timeout parsing/precedence tests, live log buffer tests, admin `ps` / `get-pipeline-log` rendering tests, compiled Go panic stack logging, and manual-intervention timeout alert behavior
- Targeted integration slice:
  - `go test -v --tags 'test integration' ./test -run 'TestBotNameHiddenCommandsUnsupportedConnector|TestHiddenPSAndGetPipelineLog|TestPipelineTimeoutWarnAndKillAlerts|TestPipelineFailureAlertIncludesTracebackExcerpt'`
  - covers hidden-command connector support/denial, live pipeline inspection, external timeout warn/kill alerts, and traceback-rich failure alerts

## Targeted Yaegi runtime repros

- `modules/yaegi-dynamic-go/yaegi_dynamic_test.go` contains a narrow repro for an interpreted-Go panic that surfaced in `plugins/go-openai-fallback` compaction work.
- Run the focused repro with: `env GOTELEMETRY=off GOCACHE=/tmp/gocache go test ./modules/yaegi-dynamic-go -run 'Test(CompiledGoMultiReturnStateAndSliceWorks|RunPluginHandlerYaegiMultiReturnPanics|RunPluginHandlerYaegiWrappedReturnWorks)$'`
- The test establishes three facts: compiled Go accepts the multi-return state/slice helper pattern, Yaegi `RunPluginHandler` panics on the same shape with `reflect.Set ... not assignable`, and a single wrapper-struct return succeeds under the same runner.

## Test Harness Scope

All test files (`*_test.go`) within the `test/` directory are gated by the `integration` build tag and leverage the single test harness defined in `test/common_test.go`. No other test harnesses have been identified in the codebase.

## Full Test Gating (JS/Lua/Sh/Go)

Large language-specific suites are gated by `RUN_FULL` so they do not run in the default `make test` path.

- To run a full suite: `RUN_FULL=js make test`, `RUN_FULL=lua make test` (or `RUN_FULL=all` to allow all full suites).
- To run the Gopherbot shell suite: `RUN_FULL=sh make test` or `TEST=ShFull make test`.
- To run Go full coverage: `RUN_FULL=go make test` or `TEST=GoFull make test`.
- `make test` sets `-run Test.*Full` when `RUN_FULL` is present to avoid running the entire suite.
- `TEST=JSFull make test` runs the JS full test without needing `RUN_FULL`.
- `TEST=LuaFull make test` runs the Lua full test without needing `RUN_FULL`.
- `TEST=ShFull make test` runs the Gopherbot shell full test without needing `RUN_FULL`.

## Local HTTP test server

- `test/http_test_server_test.go` starts a local `httptest` server for JS/Lua HTTP coverage.
- `TestJSFull` sets `GBOT_TEST_HTTP_BASEURL` so test plugins can call the local server via config.
- The server provides JSON endpoints for GET/POST/PUT plus error and timeout cases.
- The file must use the `_test.go` suffix because the `test/` directory mixes `tbot` and `tbot_test` packages; non-test files must all share one package name to compile.
