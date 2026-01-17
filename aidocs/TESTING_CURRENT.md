# Testing (Current)

Focus: integration test harness under `test/` and how tests are executed.

## How tests are discovered and run

- Integration tests are gated by build tags like `//go:build integration` in files such as `test/common_test.go` and `test/bot_integration_test.go`.
- `make test` runs `go test ... ./test` with tags `test integration netgo osusergo static_build` and optional `TEST` filter (see `Makefile` target `test`).
- The test harness depends on test-only event emission: `bot/emit_testing.go` is built with `//go:build test`, while `bot/emit_noop.go` is used when the `test` tag is not set.

## Harness entrypoints

- `setup()` in `test/common_test.go` configures `GOPHER_PROTOCOL=test` (via `init()` in the same file) and starts the bot via `StartTest()`.
- `StartTest()` is defined in `bot/start_t.go` (only built with `test` tag). It initializes the bot, selects the connector from `currentCfg.protocol`, and runs `run()` (see also `bot/bot_process.go`).
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

## Representative test suites

- Core bot behavior and message matching: `test/bot_integration_test.go` (e.g., `TestBotName`, `TestMessageMatch`).
- Memory tests: `test/memory_integration_test.go`.
- Lists plugin behavior: `test/lists_integration_test.go`.

## Test Harness Scope

All test files (`*_test.go`) within the `test/` directory are gated by the `integration` build tag and leverage the single test harness defined in `test/common_test.go`. No other test harnesses have been identified in the codebase.
