# Impact Surface Report

## 1) Change Summary

- Slice name:
  - AI chat memory lifecycle redesign
- Goal:
  - move AI conversation state from short-term memory API to long-term datum API
  - add inactivity-based retention pruning
  - replace naive history truncation with compaction model
- Out of scope:
  - changing extension API signatures
  - changing core username-based authorization semantics
  - changing connector identity contracts beyond existing behavior

## 2) Subsystems Affected (with file anchors)

- Plugin:
  - `plugins/go-openai-fallback/ai.go`
  - `plugins/go-openai-fallback/ai_test.go`
- Engine brain API usage points:
  - `bot/brain.go` (datum and short-term memory APIs consumed by plugin)
  - `robot/robot.go` (`CheckoutDatum`, `UpdateDatum`, `DeleteDatum`)
- Scheduled jobs/config:
  - `conf/robot.yaml` / robot custom equivalents (`ScheduledJobs`)
  - potential new prune job under `jobs/` or `gojobs/`
- Documentation:
  - `ENHANCEMENTS-v3.md`
  - `devdocs/ai-chat.md`
  - this `aidocs` package

## 3) Current Behavior Anchors

- Startup/order anchors:
  - no startup ordering changes required for initial slices
- Routing/message-flow anchors:
  - AI request/streaming path remains in `plugins/go-openai-fallback/ai.go`
- Identity/authorization anchors:
  - engine remains username-authoritative
- Connector behavior anchors:
  - no connector contract changes required for this memory-lifecycle work

## 4) Proposed Behavior

- What changes:
  - AI conversation persistence uses long-term datums with index
  - prune-by-inactivity job deletes old conversation datums
  - context compaction preserves summary + recent turns
- What does not change:
  - extension method signatures
  - core security model (username-authoritative)
  - connector trust boundaries
  - streaming response UX model

## 5) Invariant Impact Check

- Startup determinism preserved?:
  - yes (prune runs as normal scheduled job)
- Explicit control flow preserved?:
  - yes
- Shared auth/policy remains in engine flows?:
  - yes
- Permission checks remain username-based?:
  - yes
- Connector ordering guarantees preserved?:
  - yes
- Config precedence still explicit?:
  - yes
- Multi-connector isolation preserved (if applicable)?:
  - yes

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - low; scheduled prune runs post-start like other jobs
- Config loading/merge/precedence impact:
  - moderate; plugin gets additional config fields
- Execution ordering impact:
  - low; prune job should be bounded and idempotent
- Resource lifecycle impact:
  - moderate; additional brain I/O for index + pruning

## 7) Concurrency Risks

- Shared state touched:
  - conversation datum and shared index datum
- Locking/channel/event-order assumptions:
  - datum lock tokens and update flow must be respected
- Race/deadlock/starvation risks:
  - index update contention if many pipelines update simultaneously
- Mitigations:
  - keep index updates short
  - bounded prune batch sizes
  - retry-on-lock-expired flow

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - plugin behavior changes internally; config additions optional/defaulted
- Behavior changes for operators/users:
  - conversations persist with explicit retention controls
- Migration/fallback plan:
  - transitional read fallback from legacy state where practical
  - no required `UPGRADING-v3.md` updates for docs-only planning phase

## 9) Validation Plan

- Focused tests:
  - plugin state read/write/index/prune/compaction unit tests
- Broader regression tests:
  - `go test ./plugins/go-openai-fallback`
  - targeted connector rendering tests unchanged
- Manual verification steps:
  - run Clu in Slack and validate continuity + prune behavior

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
  - not required in first implementation slices unless scheduler semantics change
- `aidocs/COMPONENT_MAP.md` updates:
  - add prune job path once implemented
- Connector doc updates:
  - not required for storage lifecycle changes
- Other docs:
  - update `ENHANCEMENTS-v3.md` status when slices ship
  - keep `devdocs/ai-chat.md` in sync with final implementation details

## 11) Waiver (if applicable)

- Waived by:
  - n/a
- Reason:
  - n/a
- Scope limit:
  - n/a

