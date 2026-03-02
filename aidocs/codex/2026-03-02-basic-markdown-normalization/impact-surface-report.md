# Impact Surface Report — Codex Session BasicMarkdown Normalization

## 1) Change Summary

- Slice name: Codex app-server outbound message formatting alignment
- Goal: Emit Codex session thread responses as `BasicMarkdown` and normalize Codex text payloads so connector renderers receive stable, control-character-safe content.
- Out of scope:
  - Changing inbound command parsing (`start-codex`, `end-session`, etc.)
  - Altering identity/authorization/session ownership logic
  - Modifying connector-specific markdown renderers

## 2) Subsystems Affected (with file anchors)

- Files expected to change:
  - `bot/codex_session.go`
  - `bot/codex_session_test.go`
  - `aidocs/codex/2026-03-02-basic-markdown-normalization/impact-surface-report.md` (this report)
- Key functions/types:
  - `codexSendThreadMessage`
  - `codexExtractEventText`
  - `codexExtractCompletedAgentText`
  - new codex-to-BasicMarkdown normalization helper(s)

## 3) Current Behavior Anchors

- Startup/order anchors:
  - Codex sessions start/stop at runtime via built-in commands; no startup sequence changes (`bot/bot_process.go`, `bot/codex_session.go`).
- Routing/message-flow anchors:
  - Codex responses are emitted via `SendProtocolChannelThreadMessage` from `codexSendThreadMessage`.
  - Current message format is `robot.Variable`.
- Identity/authorization anchors:
  - Session control is limited to owner/admin via `codexUserMayControlSession`.
- Connector behavior anchors:
  - Connector markdown rendering is activated when format is `robot.BasicMarkdown`; currently bypassed by `robot.Variable`.

## 4) Proposed Behavior

- What changes:
  - Outbound Codex thread messages use `robot.BasicMarkdown`.
  - Codex event text is normalized before send (strip ANSI escape/control artifacts, normalize line endings, remove null/control bytes that break formatting).
  - Normalization preserves intended markdown/code content as much as possible.
- What does not change:
  - Session lifecycle, RPC call sequencing, approval handling, ownership checks, and thread routing keys.

## 5) Invariant Impact Check

- Startup determinism preserved?: Yes
- Explicit control flow preserved?: Yes
- Shared auth/policy remains in engine flows?: Yes
- Permission checks remain username-based?: Yes
- Connector ordering guarantees preserved?: Yes (single outbound call per completed turn, same path)
- Config precedence still explicit?: Yes (no config changes)
- Multi-connector isolation preserved (if applicable)?: Yes (protocol-scoped connector lookup unchanged)

## 6) Cross-Cutting Concerns

- Startup sequencing impact: None
- Config loading/merge/precedence impact: None
- Execution ordering impact:
  - No new async queues; normalization is synchronous in existing send path.
- Resource lifecycle impact:
  - No new long-lived goroutines/resources.

## 7) Concurrency Risks

- Shared state touched:
  - Existing session routing and RPC event stream only.
- Locking/channel/event-order assumptions:
  - No changes to session registry locks/channels.
- Race/deadlock/starvation risks:
  - Low; normalization helpers are pure functions.
- Mitigations:
  - Add unit tests around normalization edge cases.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - Codex thread responses will be rendered through connector BasicMarkdown logic instead of variable/raw-ish rendering.
- Behavior changes for operators/users:
  - Less terminal/control-sequence noise; improved cross-connector formatting consistency.
- Migration/fallback plan:
  - No migration required.

## 9) Validation Plan

- Focused tests:
  - `go test ./bot -run Codex -count=1`
  - new unit tests for normalization helper(s).
- Broader regression tests:
  - `go test ./bot -count=1`
- Manual verification steps:
  - Start a Codex session and verify markdown/code snippets render cleanly in threaded replies without control-sequence artifacts.

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates: Not required (no startup/control-flow changes)
- `aidocs/COMPONENT_MAP.md` updates: Not required (no component movement)
- Connector doc updates: Not required (connector semantics unchanged)
- Other docs:
  - Keep this impact report in `aidocs/codex/` for traceability.

## 11) Waiver (if applicable)

- Waived by: N/A
- Reason: N/A
- Scope limit: N/A
