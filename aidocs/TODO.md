# AI Docs TODO Backlog

This file tracks cross-cutting architecture/documentation TODO items that do not yet have a dedicated slice package.

## Open TODOs

- Make thread subscription expiration configurable instead of fixed constant:
  Current behavior uses `threadMemoryDuration = 7 * 24h` in `bot/brain.go`, and thread subscriptions are expired by `expireSubscriptions` in `bot/subscribe_thread.go`.
  This affects long-running AI thread continuity after inactivity when using subscription-based routing.
  Candidate direction:
  add a config value in `robot.yaml` for thread subscription TTL (and possibly a separate TTL for ephemeral thread memories), defaulting to current behavior.
- User validation
  - Add ValidatedUser flag to ConnectorMessage struct; meaning: the connector has a mapping entry from internalID to username and can vouch for the username being accurate. Connector guarantees that no message will be delivered for a username in it's usermap where the internalID of the user doesn't match.
  - Update engine incoming message handing of "IgnoreUnlistedUsers: true" - a message will be dropped if any of these is true:
    - ValidatedUser is unset
    - ValidatedUser is set but username not in directory
  - When IgnoreUnlistedUsers is false, if the incoming message has a username in the directory but it's not validated, the message will be dropped
  - Create a new admin-only built-in "validate user" plugin; usage:
    - Validated Admin user (incoming message for admin user and ValidatedUser set) in DM or hidden message sends the "validate user \<username\>" command, e.g. "validate user joe"
    - Robot replies with a 7-digit OTP code valid for ~30s (stored in in-memory non-persistent map, expired by brain tick when age > 30s)
    - Admin gives code to the user, who sends a message to the robot by DM or hidden message
    - When the message is seen by incoming message, and message length is 7, and the message was a DM to the robot or hidden message, the engine checks the validation map for the code; if the code matches an unexpired entry, the robot sends a DM to the admin who requested it "User validation received: \<protocol\> user '\<user\>' has internal ID '\<ID\>'" e.g. "User validation recieved: googlechat user 'parsley' has internal ID 'users/1234567'", where "parsley" came from the original validate user request.
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
