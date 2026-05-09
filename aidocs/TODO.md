# AI Docs TODO Backlog

This file tracks cross-cutting architecture/documentation TODO items that do not yet have a dedicated slice package.

## Open TODOs

- [x] Replace the `go test ./test` integration harness with a process-backed
  `gopherbot-integration` workflow:
  - Process-backed integration suites now live as readable YAML under
    `integration/suites/data/`.
  - The compiled-in suite definitions were removed from normal integration
    execution; the legacy `test/` package remains only as a compatibility
    fallback and should not be the AI/operator default.
  - MCP support runs suites through file-backed logs/results instead of
    streaming full output into model context.
  - Suite metadata now supports targeted selectors for subsystem, tag, runtime,
    and tier.
- [ ] Add privsep-only integration suites that require a real setuid/setgid
  integration binary. This remains intentionally separate from the normal
  process-backed suite because it requires host-level install state.
- [ ] Run and record a final clean-build full integration pass before tagging
  2.9.0:
  - Build with `make mcp integration-build`.
  - Run the MCP `run_integration_suite` tool for `all`.
  - Also spot-check targeted selectors such as `subsystem:pipeline`,
    `subsystem:secrets`, `subsystem:routing`, `subsystem:help`,
    `subsystem:security`, `runtime:go`, `runtime:lua`, `runtime:js`, and
    `runtime:sh`.
- [ ] Perform pre-2.9.0 pilot robot migration dry-runs:
  - Search each selected custom robot for `{{ decrypt` and move remaining
    encrypted values into custom-only `conf/variables/*.yaml` files.
  - Validate `GOPHER_ENVIRONMENT` startup behavior for development and
    production-like environments.
  - Confirm primary/secondary connector startup, reload, identity mapping, and
    hidden-command support against real configs.
  - Treat pilot findings as bugfix/UX work only unless a critical configuration
    break is discovered.
- [ ] Make thread subscription expiration configurable instead of fixed constant:
  Current behavior uses `threadMemoryDuration = 7 * 24h` in `bot/brain.go`, and thread subscriptions are expired by `expireSubscriptions` in `bot/subscribe_thread.go`.
  This affects long-running AI thread continuity after inactivity when using subscription-based routing.
  Candidate direction:
  add a config value in `robot.yaml` for thread subscription TTL (and possibly a separate TTL for ephemeral thread memories), defaulting to current behavior.
- [x] User validation
  - Added `ValidatedUser` to `ConnectorMessage`; connectors now explicitly mark whether an inbound canonical username is vouched for.
  - Updated engine handling of `IgnoreUnlistedUsers` so validated canonical identity is required for directory-gated inbound policy.
  - Added admin-only built-in `validate user <username>` with a short-lived 7-digit code and pre-pipeline DM/hidden consume path for returning protocol-local internal IDs to the requesting admin.
- [x] Update connectors to take a Reload method that, at the very least, updates the list of Validated Users.
- [x] Make external interpreter failures also send error output to the operator
  channel like built-in interpreters:
  pipeline failure alerts now go to the plugin `DefaultJobChannel` or the job's
  configured channel, include recent live log excerpts, and are covered by
  process-backed integration suites for Go, Lua, JavaScript, and Gopherbot
  shell failure cases.
- [x] Align shipped SimpleMatcher configs with the v3 DSL contract:
  - `devdocs/SimpleMatcher.md` defines the intended semantics: `(label:...)` / `(:...)` required capturing choices, `[label:...]` / `[:...]` optional capturing choices, `{...}` optional non-capturing noise, and `/.../` required non-capturing synonyms.
  - Audited `SimpleMatcher:` entries under `conf/plugins/` and test configs so plugin handler argument indexes match the documented contract.
  - Added representative shipped-pattern argument-position coverage in `bot/simple_matcher_test.go`.

## Current Cleanup TODOs

- [x] Clean-up / wrap up googlechat connector
  - Remove new user detection and logging from googlechat connector, now superceded by new Validated user bits.
  - Adjust logging: remove logging that should never be necessary, use "debug" when appropriate for debugging and "trace" for logs that should only show when we absolutely want to see everything happening.

- [x] Define connector capability semantics for private-command help/examples:
  Private-command help now uses an engine-owned connector `Capabilities` runtime flag (`HiddenCommands`) plus connector-owned slash/private rendering through the shared `robot/` contract.
  Current behavior:
  - help metadata and builtin help only surface private slash examples when the current initialized connector reports hidden/ephemeral support
  - connectors with hidden/ephemeral support render concrete private commands such as `/clu help knock/knock` through connector-owned formatting
  - engine-owned denial/help copy uses the same concrete formatter instead of placeholder `/(bot)` text or a separate connector hint

- [ ] Reconcile SSH and terminal hidden-message parsing and `IncomingMessage` shaping:
  Current local-connector behavior still has some awkwardness and drift:
  - SSH and terminal both treat slash-prefixed input as `HiddenMessage=true`, even when it is not robot-addressed
  - only `/<botname> ...` is normalized into a bot-addressed command payload before `IncomingMessage(...)`
  - the test connector is intentionally different because the harness injects the hidden flag directly instead of parsing slash syntax
  Follow-up goals:
  - decide whether SSH and terminal should keep the current split between "hidden/private" and "robot-addressed private command", or move to a cleaner unified model
  - make the SSH and terminal parsing/path shaping consistent with each other
  - review whether `IncomingMessage.MessageText`, `HiddenMessage`, and `BotMessage` are being populated in the cleanest possible way for local private commands
  - preserve engine-owned private-command policy while reducing connector-local surprises

- [x] Revisit the connector contract around canonical usernames vs transport user IDs:
  `aidocs/CONNECTOR_CONTRACT.md` now makes the username/transport-ID boundary
  explicit, and connector-specific docs cover the major protocol behavior.
  Remaining work is deployment validation rather than contract design:
  re-check Slack, Google Chat, SSH, terminal, and test connector behavior during
  pre-2.9 pilot deployments.

- [ ] Revisit runtime bot-ID fallback handling (`getRuntimeBotID`) in the engine:
  The current runtime bot-ID helpers are useful for engine-owned bot identity lookups, but they are tempting as a generic fallback for connector self-message detection.
  Follow-up goals:
  - keep self-message shaping connector-authoritative unless we intentionally redefine that contract
  - document which engine paths may legitimately consult runtime bot IDs
  - decide whether `getRuntimeBotID` should remain a narrow lookup helper or evolve into a more explicit contract with clearer boundaries
