# Slice Plan: AI Chat Memory Lifecycle

## Implementation Status

- Slice 0: complete (docs and planning artifacts)
- Slice 1: in progress
- Slice 2: pending
- Slice 3: pending
- Slice 4: pending
- Slice 5: pending

## Slice 0: Planning and Canonical Documentation

Goal:
- establish user-facing, developer-facing, and ai-facing documentation baseline.

Scope:
- `ENHANCEMENTS-v3.md`
- `devdocs/ai-chat.md`
- `aidocs/multi-protocol/2026-03-02-ai-chat-memory-lifecycle/*`

Expected result:
- implementation can proceed with a stable contract and clear slice order.

## Slice 1: Long-Term Conversation Datum Storage + Index

Goal:
- move plugin conversation state from short-term memory API to long-term datums.

Scope:
- `plugins/go-openai-fallback/ai.go`
- `plugins/go-openai-fallback/ai_test.go`

Contract:
- define `conversation:v2` datum shape
- define/update shared conversation index datum
- read/write path uses `CheckoutDatum` + `UpdateDatum`
- delete path uses `DeleteDatum` where appropriate (conversation close/reset)

Validation:
- unit tests for load/save/index update behaviors

## Slice 2: Retention Prune Job

Goal:
- enforce inactivity-based retention using scheduled job cron.

Scope:
- new prune job (Go preferred) under `jobs/` or `gojobs/`
- plugin config additions for retention parameters
- default config wiring in `conf/` and `robot.skel` as needed

Contract:
- prune by `UpdatedAt` older than `RetentionDays`
- prune batch is bounded
- delete conversation datum then remove index entry
- no plugin-local schedule key; job schedule remains in `ScheduledJobs`

Validation:
- unit tests for prune selection/deletion/index cleanup
- manual run validation via job command

## Slice 3: Deterministic Compaction

Goal:
- replace naive oldest-turn dropping with summary + recent-window compaction.

Scope:
- `plugins/go-openai-fallback/ai.go`
- `plugins/go-openai-fallback/ai_test.go`

Contract:
- preserve recent N exchanges verbatim
- compact older history into structured summary
- deterministic fallback always available

Validation:
- tests for compaction trigger and summary/recent window correctness

## Slice 4: Optional Model-Assisted Compaction

Goal:
- improve summary quality with optional AI-assisted compaction.

Scope:
- plugin compaction flow and config toggles

Contract:
- best-effort enhancement only
- fallback to deterministic compaction on failure
- never block user response path on compaction failure

Validation:
- tests for fallback paths and feature flag behavior

## Slice 5: Hardening and Final Documentation

Goal:
- finalize validation, config defaults, and status updates.

Scope:
- tests in plugin package
- `ENHANCEMENTS-v3.md` status updates (`In Progress` -> `Shipped` where applicable)
- `devdocs/ai-chat.md` implementation details synced to actual code

Validation:
- `go test ./plugins/go-openai-fallback`
- broader regressions as needed

## Suggested Validation Sequence

1. `go test ./plugins/go-openai-fallback`
2. focused connector tests if formatting path changes
3. manual Slack/SSH QA with Clu
4. full regression pass before merge
