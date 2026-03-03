# AI Chat Memory Lifecycle (Design)

## Context

The previous AI fallback implementation stored conversation state with short-term memory methods (`Remember` / `Recall`).
That was useful for initial behavior but did not provide explicit retention lifecycle, indexable pruning, or robust long-run compaction semantics.

At the same time, user workflows rely on AI chat continuity:
- brainstorming
- rubber-ducking
- code generation and analysis

The design must balance continuity with bounded storage and bounded token cost.

## Design Decision

Adopt long-term datum-backed conversation storage with:
1. explicit conversation state datum
2. explicit conversation index datum
3. prune-by-inactivity job using scheduled cron
4. context compaction using summary + recent-window model

## Core Contracts

### Storage contract

Conversation state is stored with datum APIs:
- `CheckoutDatum(...)`
- `UpdateDatum(...)`
- `DeleteDatum(...)`

Short-term memory APIs are no longer the primary storage mechanism for AI conversation state.
They remain as legacy migration fallback only.

### Keying contract

DM conversations:
- keyed by canonical username
- cross-protocol continuity for same user identity

Thread conversations:
- keyed by protocol + channel + thread ID
- avoids thread ID collisions across connectors

### Retention contract

Retention is based on `UpdatedAt` (most recent conversation activity).

Pruning cadence is controlled by normal scheduled job cron strings (`ScheduledJobs`), not a plugin-local schedule key.

### Compaction contract

When context exceeds thresholds:
- preserve a compact summary of older history
- keep recent exchanges verbatim
- avoid blind "drop oldest turn" behavior as primary strategy

## Data Model (Proposed)

### Conversation datum (`v2`)

Key format:
- `openaifallback:conversation:v2:<sha1(conversation-id)>`

Suggested fields:
- `Version`
- `ConversationID`
- `Scope` metadata (dm/thread + protocol/channel/thread)
- `Profile`
- `Owner`
- `Summary`
- `RecentExchanges`
- `Pending`
- `Processed`
- `TokenEstimate`
- `CreatedAt`
- `UpdatedAt`

### Index datum (`v1`)

Key:
- `openaifallback:conversation:index:v1`

Suggested fields:
- `Version`
- `Conversations` map of ID -> metadata:
  - conversation datum key
  - `UpdatedAt`
  - scope type

## Prompt Assembly Strategy

Inference payload should assemble context in this order:
1. system prompt
2. compact summary (older conversation state)
3. recent verbatim exchanges
4. queued pending user messages
5. current user message

## Compaction Strategy

### Stage 1 (deterministic)

Always available.
When thresholds are exceeded:
- fold older exchanges into structured summary sections
- preserve recent N exchanges verbatim

Summary should preserve:
- goals/intents
- decisions made
- constraints/assumptions
- unresolved questions
- named artifacts (files/branches/services/commands)

### Stage 2 (optional model-assisted)

Best-effort enhancement.
Use AI to rewrite older history into higher-quality structured summary.
If it fails, keep deterministic fallback behavior.

## Config Direction

Implemented plugin config fields:
- `MaxRecentExchanges`
- `CompactionTriggerTokens`
- `SummaryBudgetTokens`
- `EnableModelCompaction`

Implemented prune job config fields (`go-openai-prune`):
- `RetentionDays`
- `MaxDeletesPerRun`
- `DryRun`

Not needed:
- `PruneSchedule` (use normal cron `ScheduledJobs`)

## Invariants

Must preserve:
1. extension API signatures and behavior compatibility
2. username-authoritative security model
3. connector-local transport ID handling
4. streaming UX responsiveness

Must not introduce:
1. unbounded growth without retention controls
2. hard dependency on model-assisted compaction for correctness

## Operational Failure Handling

1. Conversation read/write failure:
   - degrade to reduced context
   - keep user interaction path alive
2. Compaction failure:
   - fallback to deterministic strategy
3. Prune failure on a subset:
   - skip failed entries
   - retry next scheduled run

## Documentation Intent

This design is the implementation contract for this workstream.
User-facing summary lives in:
- `ENHANCEMENTS-v3.md`

Human developer implementation details live in:
- `devdocs/ai-chat.md`
