# AI Chat Architecture (v3)

This document describes the current `go-openai-fallback` implementation and the memory lifecycle used for durable AI chat context.

Primary implementation target:
- `plugins/go-openai-fallback/ai.go`

Related jobs/config:
- `jobs/go-openai-prune/go_openai_prune_job.go`
- `conf/jobs/go-openai-prune.yaml`
- `conf/plugins/openai-fallback.yaml`

## Runtime Flow

For each inbound AI fallback event, the plugin:
1. builds conversation context (DM or thread scope)
2. loads conversation state from long-term datum storage
3. marks message IDs as processed and queues/merges pending messages
4. queries OpenAI with streaming responses
5. stores updated conversation state back to datum storage

Interactive path remains resilient: if compaction/prune helpers fail, the plugin falls back to deterministic/local behavior and continues replying.

## Storage Contract

Conversation state is stored in long-term datums via:
- `CheckoutDatum(...)`
- `UpdateDatum(...)`
- `DeleteDatum(...)`

Legacy short-term state (`Remember`/`Recall`) is only used as a one-time fallback read path; on successful read it is migrated into datum storage.

### Conversation identity and keys

Conversation IDs:
- DM: `dm:<username>` (cross-protocol continuity by canonical username)
- Thread: `thread:<protocol>:<channel>:<thread-id>`

Datum key:
- `openaifallback:conversation:v2:<sha1(conversation-id)>`

Index datum key:
- `openaifallback:conversation:index:v1`

The index maps conversation ID -> `{key, updated_at}` for retention pruning.

## Conversation State

Current stored state includes:
- `profile`
- `owner`
- `summary` (compacted older context)
- `exchanges` (recent verbatim exchanges)
- `pending` (queued inbound messages)
- `processed` (bounded dedupe list)
- `in_progress`
- `tokens`
- `updated_at`

Implementation limits:
- pending queue cap: 24 (`maxPendingMessages`)
- processed ID cap: 48 (`maxProcessedMessages`)
- stored exchanges hard cap: 48 (`maxStoredExchanges`)

## Prompt Assembly

Inference payload message order:
1. system prompt
2. compact summary (if present)
3. recent exchanges
4. pending queued messages
5. current message (`"<user> says: ..."`)

Output format is `BasicMarkdown`.

## Compaction

### Deterministic compaction (always available)

When compaction triggers:
- keep `MaxRecentExchanges` verbatim
- fold older exchanges into `summary`

Config knobs:
- `CompactionTriggerTokens`
- `MaxRecentExchanges`
- `SummaryBudgetTokens`

### Optional model-assisted compaction

If `EnableModelCompaction: true`, the plugin attempts to refine deterministic summary text with a non-streaming OpenAI call.

Important behavior:
- deterministic compaction runs first
- model-assisted compaction is best-effort only
- failures keep deterministic summary (no user-path failure)

## Retention Pruning

Pruning is done by scheduled job `go-openai-prune`, not by plugin-local schedule settings.

Job behavior:
1. read index datum
2. select entries older than `RetentionDays`
3. delete conversation datums (bounded by `MaxDeletesPerRun`)
4. remove successfully deleted entries from index

Job config:
- `RetentionDays`
- `MaxDeletesPerRun`
- `DryRun`

Schedule via `ScheduledJobs` cron in robot config.

## Testing

Primary slice tests live in:
- `plugins/go-openai-fallback/ai_test.go`
- `jobs/go-openai-prune/go_openai_prune_job_test.go`

Recommended validation command set:
1. `go test ./plugins/go-openai-fallback`
2. `GOCACHE=/tmp/gocache go test ./jobs/go-openai-prune`
3. manual Slack/SSH validation on a real robot config
