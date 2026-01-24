# Testing (Current)

Focus: integration test harness under `test/` and how tests are executed.

## How tests are discovered and run

- Integration tests are gated by build tags like `//go:build integration` in files such as `test/common_test.go` and `test/bot_integration_test.go`.
- `make test` runs `go test ... ./test` with tags `test integration netgo osusergo static_build` and optional `TEST` filter (see `Makefile` target `test`).
- The test harness depends on test-only event emission: `bot/emit_testing.go` is built with `//go:build test`, while `bot/emit_noop.go` is used when the `test` tag is not set.

## Harness entrypoints

- `setup()` in `test/common_test.go` starts the bot via `StartTest()` and relies on the test configs to choose the `test` protocol.
- `StartTest()` is defined in `bot/start_t.go` (only built with `test` tag). It locates the repo root by walking up until `conf/robot.yaml` is found, `chdir`s there so startup mode resolves to `test-dev`, then initializes the bot, selects the connector from `currentCfg.protocol`, and runs `run()` (see also `bot/bot_process.go`).
- The test connector is registered in `connectors/test/init.go` (`bot.RegisterConnector("test", Initialize)`), and its runtime loop lives in `connectors/test/connector.go` (`(*TestConnector).Run`).

## setup / teardown / testcases flow

- `setup(cfgdir, logfile, t)` sets `GOPHER_ENCRYPTION_KEY`, wires `testc.ExportTest.Test`, and calls `StartTest()` (see `test/common_test.go`).
- `teardown(t, done, conn)` sends a `quit` message and waits on `done`, then checks emitted events from `GetEvents()` (see `test/common_test.go` and `bot/emit_testing.go`).
- `testcases(t, conn, tests)` drives each test by:
  - Clearing startup events with `GetEvents()` before sending the message (see `test/common_test.go`).
  - Sending the message through `conn.SendBotMessage` (type `testc.TestMessage` in `connectors/test/connector.go`).
  - Matching expected replies using regex on `TestMessage.Message` and strict equality on user/channel/threaded fields (see `test/common_test.go`).
  - Comparing observed `[]Event` from `GetEvents()` with expected events (event type `Event` in `bot/events.go`).

## Test case structure

- `testItem` in `test/common_test.go` defines a case as:
  - `user`, `channel`, `message`, `threaded` (input fields).
  - `replies []TestMessage` where `TestMessage.Message` is a regex to match output (type `TestMessage` in `test/common_test.go`).
  - `events []Event` expected for the interaction (type `Event` in `bot/events.go`).
  - `pause` (milliseconds) to sleep between cases (see `test/common_test.go`).
- Hidden messages are signaled by a leading `/` in `message` and transformed before sending (see `test/common_test.go`).

## Protocol selection for tests

- The integration configs under `test/*/conf/robot.yaml` set `Protocol: {{ env "GOPHER_PROTOCOL" | default "test" }}` so tests use the test connector by default without setting env vars in Go code.
- When running a config interactively (e.g., `cd test/membrain` + `gopherbot`), set `GOPHER_PROTOCOL=terminal` (via `private/environment`) to exercise the terminal connector.

## `make testbot` (interactive test helper)

- `make testbot` builds `gopherbot` with the `test` build tag, enabling test-only behavior like event capture (`bot/emit_testing.go`) and terminal send formatting (`bot/term_sendmessage_tbot.go`).
- In the terminal connector, pressing `<enter>` with no input prints buffered events from `GetEventStrings()` (see `bot/term_connector.go` and `bot/emit_testing.go`).

## Representative test suites

- Core bot behavior and message matching: `test/bot_integration_test.go` (e.g., `TestBotName`, `TestMessageMatch`).
- Memory tests: `test/memory_integration_test.go`.
- Lists plugin behavior: `test/lists_integration_test.go`.

## Test Harness Scope

All test files (`*_test.go`) within the `test/` directory are gated by the `integration` build tag and leverage the single test harness defined in `test/common_test.go`. No other test harnesses have been identified in the codebase.

## Full Test Gating (JS/Lua/Go)

Large language-specific suites are gated by `RUN_FULL` so they do not run in the default `make test` path.

- To run a full suite: `RUN_FULL=js make test`, `RUN_FULL=lua make test` (or `RUN_FULL=all` to allow all full suites).
- `make test` sets `-run Test.*Full` when `RUN_FULL` is present to avoid running the entire suite.
- `TEST=JSFull make test` runs the JS full test without needing `RUN_FULL`.
- `TEST=LuaFull make test` runs the Lua full test without needing `RUN_FULL`.

## Local HTTP test server

- `test/http_test_server_test.go` starts a local `httptest` server for JS/Lua HTTP coverage.
- `TestJSFull` sets `GBOT_TEST_HTTP_BASEURL` so test plugins can call the local server via config.
- The server provides JSON endpoints for GET/POST/PUT plus error and timeout cases.
- The file must use the `_test.go` suffix because the `test/` directory mixes `tbot` and `tbot_test` packages; non-test files must all share one package name to compile.
