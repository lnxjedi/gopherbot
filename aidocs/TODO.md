# AI Docs TODO Backlog

This file tracks cross-cutting architecture/documentation TODO items that do not yet have a dedicated slice package.

## Open TODOs

- Make thread subscription expiration configurable instead of fixed constant:
  Current behavior uses `threadMemoryDuration = 7 * 24h` in `bot/brain.go`, and thread subscriptions are expired by `expireSubscriptions` in `bot/subscribe_thread.go`.
  This affects long-running AI thread continuity after inactivity when using subscription-based routing.
  Candidate direction:
  add a config value in `robot.yaml` for thread subscription TTL (and possibly a separate TTL for ephemeral thread memories), defaulting to current behavior.

## Current Cleanup TODOs

- [x] Restore environment-specific encryption key behavior for `GOPHER_ENVIRONMENT`, but with safe fallback:
  If `custom/binary-encrypted-key.<environment>` is missing, log a warning and fall back to the usual `custom/binary-encrypted-key` so development can reuse production-era encrypted secrets unless a separate per-environment key is intentionally created.

- [x] Enumerate branch changes made during the current AI fallback / wrong-channel / help UX work:
  Build a concise, decision-oriented list of what changed on this branch so we can separate "definitely merge", "maybe keep", and "back out or rethink".
  Current inventory:
  - wrong-channel detection moved into the engine, with improved availability/location hints and updated tests
  - alias fallback/help work added engine help metadata and deterministic built-in fallback ranking
  - builtin help/fallback UX changed, including `help <keyword> brief` and smarter deterministic recovery behavior
  - OpenAI fallback fixes landed for compaction, `max_completion_tokens`, and a Yaegi multi-return workaround/repro
  - multiprocess/interpreter cleanup landed, including more generic interpreter RPC naming and related docs/tests
  - startup/config fixes landed, including the prune-job config fix, `GOPHER_ENVIRONMENT` encryption-key fallback behavior, and the integration-test fixture cleanup needed after the new fallback defaults

- [ ] Decide how to merge the changes we definitely want into `v3-dev`:
  After enumerating the branch work, choose a clean merge strategy for `v3-dev` and identify anything that should stay behind for now.

- [ ] Revisit AI-assisted UX for mistyped commands and syntax mistakes:
  Now that wrong-channel detection has moved into the engine, re-evaluate where AI still adds value for alias fallback, especially around syntax recovery, missing arguments, and likely intended command help.

- [ ] Take a hard look at help-system UX:
  Re-evaluate quick help, `help <keyword>`, brief help, channel/context guidance, and how fallback/help should compose so the user gets the most useful next step with the least noise.

- [x] Define connector capability semantics for hidden-command help/examples:
  Hidden-command help now uses an engine-owned connector `Capabilities` registration flag (`HiddenCommands`) plus connector-owned hidden-help rendering through the shared `robot/` contract.
  Current behavior:
  - help metadata and builtin help only surface hidden examples when the current connector is registered as supporting hidden commands
  - SSH/terminal/test render exact `/(bot) ...` hidden examples through connector-owned formatting
  - Slack reports hidden-command support but only provides a generic slash-command hint, because the configured slash command name is not the bot name and is not inferred by help
