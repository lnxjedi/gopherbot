# Impact Surface Report

## 1) Change Summary

- Slice name: `prompt-timeout-ssh-terminal-slice5`
- Goal: Keep default prompt wait behavior for chat connectors, while extending prompt wait timeout to 42 minutes for interactive local connectors (`ssh`, `terminal`) when prompt is issued by built-in/interpreted task paths; also make in-progress prompts interrupt promptly during robot shutdown.
- Out of scope:
  - New YAML/config knobs for prompt timeout
  - Per-plugin prompt timeout overrides
  - Connector-specific prompt queue redesign

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/replyprompt.go`
  - `bot/bot_process.go`
  - `bot/replyprompt_test.go` (new)
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/PIPELINE_LIFECYCLE.md`
  - `aidocs/EXTENSION_API.md` (if prompt semantics text is updated)
  - `aidocs/SSH_CONNECTOR.md`
- Key functions/types/symbols:
  - `Robot.promptInternal(...)`
  - `replyMatcher`, `replyWaiter`, global `replies` map
  - `stop()` and startup init path in `initBot()`

## 3) Current Behavior Anchors

- Startup/order anchors:
  - `initBot()` initializes process-wide state.
  - `stop()` currently waits for pipelines (`state.Wait()`) before connector runtime shutdown.
- Routing/message-flow anchors:
  - Incoming messages first check `replies` waiter map in `worker.handleMessage()`.
  - Prompt waits are keyed by `user/channel/thread` matcher and resolved in FIFO-with-retry style.
- Identity/authorization anchors:
  - No identity model changes in this slice; prompt match key remains normalized username + channel + thread.
- Connector behavior anchors:
  - Prompt send uses protocol-aware connector send methods already.
  - Timeout is a single fixed `45s` constant regardless of connector/task type.

## 4) Proposed Behavior

- What changes:
  - Replace fixed timeout usage with a resolver that returns:
    - `45s` default for existing behavior
    - `42m` for prompts from `ssh`/`terminal` contexts when caller task path is built-in/interpreted (compiled-in Go task/plugin/job, or external `.go/.lua/.js`).
  - Add prompt-shutdown cancellation signal so pending `Prompt*` waits return immediately as `Interrupted` once shutdown starts.
  - Ensure prompt cancellation signal is reset on startup/restart lifecycle.
- What does not change:
  - Waiter keying (`user/channel/thread`) and reply matching order.
  - Retry semantics for overlapping waiters.
  - Authorization/roster/policy enforcement location.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes (unchanged)
- Connector ordering guarantees preserved?: yes (unchanged)
- Config precedence still explicit?: yes (no config changes)
- Multi-connector isolation preserved (if applicable)?: yes

No invariant is redefined in this slice.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - Need deterministic reinitialization of prompt-shutdown signal during startup/init.
- Config loading/merge/precedence impact:
  - none.
- Execution ordering impact:
  - none for dispatch ordering; prompt waits may end earlier during shutdown.
- Resource lifecycle impact (connections, goroutines, shutdown):
  - Prompt wait goroutines should not block stop flow for long timeout windows.

## 7) Concurrency Risks

- Shared state touched:
  - `replies.m` waiter map
  - new global prompt-shutdown signal state
- Locking/channel/event-order assumptions:
  - Prompt waiters must observe shutdown signal without deadlock.
  - Shutdown signal close must be one-time and race-safe.
- Race/deadlock/starvation risks:
  - Risk of close-on-closed channel if stop can execute concurrently.
  - Risk of stale canceled channel after restart if not reinitialized.
- Mitigations:
  - Guard shutdown signal with mutex + closed flag.
  - Recreate signal in startup initialization path.
  - Keep waiter cleanup logic under existing `replies` lock discipline.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - No required config changes.
- Behavior changes for operators/users:
  - For `ssh`/`terminal` with built-in/interpreted tasks, prompts can wait much longer (up to 42 minutes).
  - During shutdown, pending prompts now interrupt quickly instead of waiting for timeout.
- Migration/fallback plan:
  - No migration step needed.
  - Fallback is code rollback if prompt UX regression appears.

## 9) Validation Plan

- Focused tests:
  - Timeout resolver cases (default vs `ssh`/`terminal` + interpreter).
  - Prompt wait returns `Interrupted` on shutdown signal.
- Broader regression tests:
  - `go test ./bot`
  - if green, optional `go test ./...` for broader confidence.
- Manual verification steps:
  - Prompt flow in ssh connector with delayed response.
  - Trigger shutdown while prompt active and confirm immediate interrupt.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - add shutdown sequence note that prompt waits are interrupted before pipeline wait.
- `aidocs/COMPONENT_MAP.md` updates:
  - none expected (no boundary move).
- Connector doc updates:
  - likely note in `aidocs/SSH_CONNECTOR.md` about long interactive prompt window.
- Other docs:
  - `aidocs/PIPELINE_LIFECYCLE.md` prompt wait/shutdown cancellation behavior note.
  - `aidocs/EXTENSION_API.md` prompt timeout semantics note.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
