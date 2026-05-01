# AI Docs TODO Backlog

This file tracks cross-cutting architecture/documentation TODO items that do not yet have a dedicated slice package.

## Open TODOs

- [ ] Deprecate `Exclusive` in v3 and add a replacement `TaskLock(tag, queue bool) RetVal` API:
  - Keep the existing `Exclusive(tag, queueTask bool) bool` behavior for v3 compatibility, but document it as deprecated once `TaskLock` exists.
  - Add exactly one new `RetVal` value, `Busy`; use existing `Ok` and `Failed` for the other states.
  - `Ok` means the lock was acquired and the caller should proceed.
  - `Busy` means the lock was not acquired and the caller should exit cleanly. When `queue` is `true`, `Busy` must mean the retry was successfully queued; if queuing was requested but could not be guaranteed, return `Failed` instead.
  - `Failed` means API misuse or an internal error; the engine must emit an Error-level log with enough detail for the robot owner to diagnose the issue.
  - Treat contention as normal cooperative behavior, not an error. Reserve operator-visible failure handling for `Failed`.
- [ ] Replace the `go test ./test` integration harness with a process-backed
  `gopherbot-integration` workflow:
  - Use `aidocs/INTEGRATION_HARNESS_PLAN.md` as the design source.
  - Keep suite definitions in Go so existing tests can be ported mechanically.
  - Preserve the old `StartTest` path until coverage parity is proven.
  - [x] Add MCP support so integration suites run through file-backed logs/results
    instead of streaming full output into model context.
  - Add privsep-only suites that require a real setuid/setgid integration
    binary.
- Make thread subscription expiration configurable instead of fixed constant:
  Current behavior uses `threadMemoryDuration = 7 * 24h` in `bot/brain.go`, and thread subscriptions are expired by `expireSubscriptions` in `bot/subscribe_thread.go`.
  This affects long-running AI thread continuity after inactivity when using subscription-based routing.
  Candidate direction:
  add a config value in `robot.yaml` for thread subscription TTL (and possibly a separate TTL for ephemeral thread memories), defaulting to current behavior.
- [x] User validation
  - Added `ValidatedUser` to `ConnectorMessage`; connectors now explicitly mark whether an inbound canonical username is vouched for.
  - Updated engine handling of `IgnoreUnlistedUsers` so validated canonical identity is required for directory-gated inbound policy.
  - Added admin-only built-in `validate user <username>` with a short-lived 7-digit code and pre-pipeline DM/hidden consume path for returning protocol-local internal IDs to the requesting admin.
- [ ] Update connectors to take a Reload method that, at the very least, updates the list of Validated Users.
- [ ] Make external interpreter failures ALSO send error output to jobs channel for default connector, like built-in interpreters
- [x] Align shipped SimpleMatcher configs with the v3 DSL contract:
  - `devdocs/SimpleMatcher.md` defines the intended semantics: `(...)` required capturing, `[...]` optional capturing, `{...}` optional non-capturing noise, and `/.../` required non-capturing synonyms.
  - Audited `SimpleMatcher:` entries under `conf/plugins/` and test configs so plugin handler argument indexes match the documented contract.
  - Added representative shipped-pattern argument-position coverage in `bot/simple_matcher_test.go`.

## Current Cleanup TODOs

- [ ] Clean-up / wrap up googlechat connector
  - Remove new user detection and logging from googlechat connector, now superceded by new Validated user bits.
  - Adjust logging: remove logging that should never be necessary, use "debug" when appropriate for debugging and "trace" for logs that should only show when we absolutely want to see everything happening.

- [x] Define connector capability semantics for hidden-command help/examples:
  Hidden-command help now uses an engine-owned connector `Capabilities` runtime flag (`HiddenCommands`) plus connector-owned hidden-help rendering through the shared `robot/` contract.
  Current behavior:
  - help metadata and builtin help only surface hidden examples when the current initialized connector reports hidden-command support
  - connectors with hidden-command support render concrete hidden commands such as `/clu help knock/knock` through connector-owned formatting
  - engine-owned denial/help copy uses the same concrete hidden-command formatter instead of placeholder `/(bot)` text or a separate connector hint

- [ ] Reconcile SSH and terminal hidden-message parsing and `IncomingMessage` shaping:
  Current local-connector behavior still has some awkwardness and drift:
  - SSH and terminal both treat slash-prefixed input as `HiddenMessage=true`, even when it is not robot-addressed
  - only `/<botname> ...` is normalized into a bot-addressed command payload before `IncomingMessage(...)`
  - the test connector is intentionally different because the harness injects the hidden flag directly instead of parsing slash syntax
  Follow-up goals:
  - decide whether SSH and terminal should keep the current split between "hidden/private" and "robot-addressed hidden command", or move to a cleaner unified model
  - make the SSH and terminal parsing/path shaping consistent with each other
  - review whether `IncomingMessage.MessageText`, `HiddenMessage`, and `BotMessage` are being populated in the cleanest possible way for local hidden commands
  - preserve engine-owned hidden-command policy while reducing connector-local surprises

- [ ] Revisit the connector contract around canonical usernames vs transport user IDs:
  The current split between engine username authority, connector-local `UserMap`, inbound `ProtocolUser`, and outbound user-targeted send helpers grew organically over time and is now subtle enough to cause connector drift.
  Follow-up goals:
  - make the contract for canonical username, protocol user ID, and connector-side lookup/fallback explicit
  - decide what the engine may assume about `userProto` / `userIDProto` population from connector-local identity maps
  - document when connectors may treat a string as an already-resolved transport ID versus when they must resolve a canonical username
  - re-check Slack, Google Chat, SSH, terminal, and test connector behavior against that contract

- [ ] Revisit runtime bot-ID fallback handling (`getRuntimeBotID`) in the engine:
  The current runtime bot-ID helpers are useful for engine-owned bot identity lookups, but they are tempting as a generic fallback for connector self-message detection.
  Follow-up goals:
  - keep self-message shaping connector-authoritative unless we intentionally redefine that contract
  - document which engine paths may legitimately consult runtime bot IDs
  - decide whether `getRuntimeBotID` should remain a narrow lookup helper or evolve into a more explicit contract with clearer boundaries
