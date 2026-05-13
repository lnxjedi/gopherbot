# Testing (Current)

Focus: process-backed integration suites under `integration/suites/data/`,
the legacy compatibility harness under `test/`, and how both are executed.

Forward plan:

- The current `go test ./test` harness is being replaced by a process-backed
  `gopherbot-integration` command. See
  `aidocs/INTEGRATION_HARNESS_PLAN.md`.
- During migration, the existing `StartTest()` path remains a compatibility
  harness for legacy-only debugging, but AI automation should use the
  MCP-integrated process-backed runner by default.
- `make integration` now builds `gopherbot-integration` and prints suite-runner
  instructions. Use `make integration-legacy` for the old Go test harness.
- Process-backed suite definitions are data-driven YAML files in
  `integration/suites/data/`. Each file names one suite, its config directory,
  metadata, scripted input messages, expected replies, expected event
  sequence, and any suite-local hooks such as `before_start: test_http_server`.

## How tests are discovered and run

- Integration tests are gated by build tags like `//go:build integration` in files such as `test/common_test.go` and `test/bot_integration_test.go`.
- `make integration-legacy` runs `go test ... ./test` with tags `test integration netgo osusergo static_build` and optional `TEST` filter. This is the old compatibility harness; do not use it for normal AI validation unless explicitly requested.
- `make test` runs unit tests, then builds `gopherbot-integration` through the
  `integration` target.
- The test harness depends on test-only event emission: `bot/emit_testing.go` is built with `//go:build test`, while `bot/emit_noop.go` is used when the `test` tag is not set.

## Harness entrypoints

- `gopherbot-integration run-suite <SuiteName>` is the new process-backed
  suite-runner entrypoint. It creates an isolated directory under
  `integration/runs/`, starts a real robot process in that directory using the
  scripted test connector, prints live interaction when requested, and writes
  `result.json`, `robot.log`, and `transcript.txt`. The transcript contains the
  same `->` / `<-` interaction lines that live output prints, plus step-level
  progress such as case starts, idle waits, event drains, and expected replies.
- `gopherbot-integration run-suite` enforces a per-test timeout with
  `-case-timeout` (default `14s`). If a case or YAML flow step exceeds that
  deadline, the suite writes `goroutines.txt`, marks the failed step with
  `timed_out: true` in `result.json`, and exits the suite process hard so the
  goroutine state reflects the hang.
- After scripted cases complete, the runner sends the normal admin `bender quit`
  command in `#general`. If the robot does not exit within the same
  `-case-timeout`, the runner writes `goroutines.txt`, records a `shutdown`
  timeout failure, and exits hard.
- `gopherbot-integration` owns its top-level CLI. Running it with no arguments
  or `--help` prints integration-runner help rather than starting a robot.
  Explicit `gopherbot-integration run [gopherbot flags]` is the high-fidelity
  robot-start path for debugging.
- `gopherbot-integration run-suite all` creates one timestamped artifact
  directory for the invocation, then writes each suite under its own
  subdirectory inside that run directory. The runner prints the single
  "Results recorded in" path at the end.
- `gopherbot-integration run-suite` also accepts exact suite names, glob
  patterns such as `TestLuaFull*`, multiple selectors as separate CLI
  arguments, and comma-separated selector lists for MCP calls. Any selector
  invocation that resolves to multiple suites uses one timestamped artifact
  directory for the whole invocation.
- `gopherbot-integration run-suite` also accepts metadata selectors:
  `subsystem:<name>`, `tag:<name>`, `runtime:<name>`, and `tier:<name>`. These
  selectors are resolved by the integration binary, so MCP callers should pass
  the same selector string instead of reimplementing suite filtering.
- `gopherbot-mcp` exposes integration runner tools for AI automation:
  `list_integration_suites`, `run_integration_suite`, and
  `read_integration_result`. `run_integration_suite` builds
  `gopherbot-integration` by default, runs the selected suite with output
  redirected to artifact files, and returns a compact summary plus paths,
  including `results_root` for the common output directory. It passes
  `case_timeout_ms` through to `gopherbot-integration` when supplied; the
  default is `14000`.
- `make integration-mcp TEST=<SuiteName>` wraps `gopherbot-mcp`
  `run_integration_suite` for local/AI use. Prefer this wrapper when running
  from Codex so the already-approved `make` path is used instead of repeatedly
  requesting approval for direct integration commands.
- For AI-driven verification, prefer `gopherbot-mcp` `run_integration_suite`
  over direct `go test ./test`. This avoids streaming noisy logs into context
  and keeps integration artifacts under `integration/runs/`.
- `gopherbot-integration list-suites` lists the YAML-loaded suite registry. The
  readable source of those suites is `integration/suites/data/*.yaml`; incoming
  message users may use names like `alice` and `bob`, which the YAML loader maps
  to the test connector's configured transport IDs (`u0001`, `u0002`, etc.).
  Reply expectations use canonical usernames because replies are observed after
  connector identity normalization.
- `gopherbot-integration list-suites -json` returns suite metadata for tooling.
  `list-suites` also supports `-subsystem`, `-tag`, `-runtime`, and `-tier`
  filters for local discovery.
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

When an MCP-driven integration suite fails, check these first:

1. The compact `run_integration_suite` summary.
2. The saved `result.json` artifact.
3. The suite `runner.log` and robot `robot.log` under the artifact directory.
4. The suite config dir reported in the result.

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

Example (`ai-fallback` conversation records):

- `../gopherbot/gopherbot list | rg "aifallback:conversation|conversation:index"`
- `../gopherbot/gopherbot fetch ai-fallback:aifallback:conversation:index:v1`
- `../gopherbot/gopherbot fetch ai-fallback:aifallback:conversation:v2:<sha1>`

Notes:

- `list` returns fully namespaced datum keys (for plugins this is typically `<plugin-name>:<datum-key>`).
- `fetch` returns raw JSON payloads, which is useful for confirming actual stored shape versus expected struct shape.
- CLI memory commands initialize the configured brain provider object directly
  and do not start the normal serialized `runBrain()` loop.

## Test case structure

- Process-backed cases are YAML objects under `integration/suites/data/`:
  - `metadata.subsystems` is the curated selection vocabulary for targeted
    validation, for example `pipeline`, `authz`, `help`, `secrets`, or
    `extension-api`.
  - `metadata.tags` is a descriptive lower-kebab label set for discovery.
  - `metadata.runtimes` names extension/runtime coverage such as `go`, `js`,
    `lua`, `sh`, `python`, or `ruby`.
  - `metadata.tier` is `smoke`, `focused`, or `full`.
  - `input` contains sender, channel, text, and optional `threaded` / `hidden`.
  - `replies` contains regex-based expected bot messages plus strict
    user/channel/thread fields.
  - `events` is the ordered list of expected `bot.Event` names.
  - `replies_only: true` skips event comparison for intentionally noisy
    pipeline/admin flows.
  - `pause` accepts Go duration strings such as `150ms`.
- YAML `flow` suites cover multi-step interactions that need captures or
  interleaving, such as validation codes and admin pipeline log inspection.
- `testItem` in `test/common_test.go` defines a case as:
  - `user`, `channel`, `message`, `threaded` (input fields).
  - `replies []TestMessage` where `TestMessage.Message` is a regex to match output (type `TestMessage` in `test/common_test.go`).
  - `events []Event` expected for the interaction (type `Event` in `bot/events.go`).
  - `pause` (milliseconds) to sleep between cases (see `test/common_test.go`).
- Hidden messages are signaled by a leading `/` in `message` and transformed before sending (see `test/common_test.go`).
- The harness sets the test connector's `Hidden` flag directly; the test connector does not parse slash syntax itself. That is intentional so integration suites can simulate protocols with and without hidden-command support using the same connector.
- Hidden admin/history/job-command coverage should continue using that same mechanism rather than trying to emulate connector-local slash parsing in the test connector.
- Private/channel rule coverage uses the focused process-backed `TestPrivateRules` suite with the test connector's `hidden` flag:
  - private-capable commands remain runnable in DMs and hidden contexts even when normal public help/channel visibility is scoped elsewhere
  - non-private commands remain rejected in private contexts
  - `RequiredPrivateCommands` remain rejected in public contexts
  - `RestrictPrivateChannels: true` rejects DMs and rejects hidden invocations outside configured plugin channels while allowing hidden invocation from configured channels
  - hidden denial paths still cover connector hidden-command capability and robot-addressing requirements separately from channel-restriction policy

## Protocol selection for tests

- The integration configs under `test/*/conf/robot.yaml` set `PrimaryProtocol: {{ env "GOPHER_PROTOCOL" | default "test" }}` so tests use the test connector by default without setting env vars in Go code.
- When running a config interactively (e.g., `cd test/membrain` + `gopherbot`), set `GOPHER_PROTOCOL=terminal` (via `private/environment`) to exercise the terminal connector.

## `make testbot` (interactive test helper)

- `make testbot` builds `gopherbot` with the `test` build tag, enabling test-only behavior like event capture (`bot/emit_testing.go`) and terminal send formatting (`bot/term_sendmessage_tbot.go`).
- In the terminal connector, pressing `<enter>` with no input prints buffered events from `GetEventStrings()` (see `bot/term_connector.go` and `bot/emit_testing.go`).

## Representative test suites

- Core bot behavior and message matching: `integration/suites/data/TestBotName.yaml` and `integration/suites/data/TestMessageMatch.yaml`.
- Memory tests: `integration/suites/data/TestMemory.yaml`.
- Lists plugin behavior: `integration/suites/data/TestLists.yaml`.
- External Python/Ruby EncryptSecret coverage: `integration/suites/data/TestExternalEncryptSecret.yaml`.
- External yaegi Go full coverage: `integration/suites/data/TestGoFull.yaml`.
- Gopherbot shell full coverage: `integration/suites/data/TestShFull.yaml` plus `plugins/test/shfull.gsh`.
- JavaScript full coverage: `integration/suites/data/TestJSFull.yaml` plus `plugins/test/jsfull.js`, including the OAuth2 link/get/unlink engine API cycle.
- Admin/watchdog coverage: YAML files such as `integration/suites/data/TestHiddenPSAndGetPipelineLog.yaml`, using `test/membrain/plugins/admininspect.sh` and `test/membrain/plugins/admintimeout.sh` to exercise:
  - hidden `ps` and `get-pipeline-log`
  - timeout warn/kill operator alerts for external pipelines
  - operator-facing failure alerts with stderr/traceback excerpts

## Focused routing/help metadata checks

- `go test ./bot ./modules/yaegi-dynamic-go`
- This focused set currently covers:
  - privsep supplementary-group policy parsing and fail-closed decisions in `bot/privsep_process_test.go`
  - catch-all mode routing and selection in `bot/help_metadata_api_test.go`
  - engine-side help metadata filtering, ranking inputs, and deterministic fallback advice in `bot/help_metadata_api_test.go`
  - Yaegi symbol/runtime coverage for active Robot API methods in `modules/yaegi-dynamic-go/yaegi_dynamic_test.go`
  - shared `.yaegi-gopath` import coverage for installed (`gopherbot.internal/lib/...`) and custom (`robot.internal/lib/...`) interpreted-Go libraries

## Focused Admin/Watchdog Verification

- Bot unit package:
  - `go test ./bot`
  - includes timeout parsing/precedence tests, live log buffer tests, admin `ps` / `get-pipeline-log` rendering tests, compiled Go panic stack logging, and manual-intervention timeout alert behavior
- Targeted integration slice:
  - use `gopherbot-mcp` `run_integration_suite` for each relevant process-backed suite, for example `TestHiddenPSAndGetPipelineLog`, `TestPipelineTimeoutWarnAndKillAlerts`, and `TestPipelineFailureAlertIncludesTracebackExcerpt`
  - covers hidden-command connector support/denial, live pipeline inspection, external timeout warn/kill alerts, and traceback-rich failure alerts

## Targeted Yaegi runtime repros

- `modules/yaegi-dynamic-go/yaegi_dynamic_test.go` contains a narrow repro for an interpreted-Go panic that surfaced in `plugins/go-ai-fallback` compaction work.
- Run the focused repro with: `env GOTELEMETRY=off GOCACHE=/tmp/gocache go test ./modules/yaegi-dynamic-go -run 'Test(CompiledGoMultiReturnStateAndSliceWorks|RunPluginHandlerYaegiMultiReturnPanics|RunPluginHandlerYaegiWrappedReturnWorks)$'`
- The test establishes three facts: compiled Go accepts the multi-return state/slice helper pattern, Yaegi `RunPluginHandler` panics on the same shape with `reflect.Set ... not assignable`, and a single wrapper-struct return succeeds under the same runner.

## Test Harness Scope

Process-backed suites are loaded from `integration/suites/data/*.yaml` by
`integration/suites/yaml_loader.go` and executed by
`cmd/gopherbot-integration/main.go`. Legacy test files (`*_test.go`) within the
`test/` directory are gated by the `integration` build tag and leverage the
single compatibility harness defined in `test/common_test.go`.

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

- `integration/suites/http_server.go` starts a local `httptest` server for JS/Lua HTTP coverage when a YAML suite sets `before_start: test_http_server`.
- `TestJSFull` and `TestLuaFull` set `GBOT_TEST_HTTP_BASEURL` through that hook so test plugins can call the local server via config.
- The server provides JSON endpoints for GET/POST/PUT plus error and timeout cases.
