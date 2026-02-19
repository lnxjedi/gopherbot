# Proposal 1 Coding Handoff

## Goal

Implement Proposal 1 from the help-metadata analysis:

1. Remove `Helptext` from v3 command metadata.
2. Keep `Keywords` optional.
3. Preserve good deterministic help/fallback ranking without AI.

This document is written for a lower-reasoning coding run that should execute slices in order with minimal ambiguity.

## Scope

In scope:

- Help metadata collection and ranking logic in `bot/builtins.go`.
- Command matcher schema in `bot/tasks.go`.
- Validation tests and help-ranking tests.
- Default config and plugin default YAML blobs in this repository.
- Migration docs.

Out of scope:

- Connector behavior changes.
- Startup flow changes.
- Authorization semantics changes.
- New UX features (Proposal 2 / topic-level help).

## Architectural Guardrails

Must remain true after each slice:

1. Startup ordering unchanged (`aidocs/STARTUP_FLOW.md`).
2. Routing order unchanged (`aidocs/PIPELINE_LIFECYCLE.md`).
3. Username-based auth and group filtering unchanged (`bot/authorize.go`, `bot/builtins.go`).
4. No connector-specific policy logic introduced.
5. Deterministic ranking and deterministic tie-break ordering preserved.

## Impact Surface Report

### 1) Change Summary

- Slice name: `proposal1-help-metadata-slimming`
- Goal: Reduce author burden by deleting low-value `Helptext` metadata and making `Keywords` optional with deterministic auto-indexing.
- Out of scope: Proposal 2 (`HelpTopics`) and broader UX redesign.

### 2) Subsystems Affected (with file anchors)

- `bot/builtins.go`
  - `collectHelpCommandMetadata`
  - `scoreHelpCommandMatch`
  - `rankHelpMatches`
  - `renderHelpEntry`
- `bot/tasks.go`
  - `InputMatcher` struct
- `bot/builtins_help_metadata_test.go`
- `bot/taskconf_commands_test.go`
- Config defaults and plugin Configure blobs:
  - `conf/plugins/*.yaml`
  - `plugins/go-lists/lists.go`
  - `plugins/go-links/links.go`
  - `plugins/go-knock/knock.go`
  - `plugins/go-new-robot/new_robot.go`
  - `goplugins/duo/duo.go`
- Docs:
  - `UPGRADING-v3.md`
  - `aidocs/PIPELINE_LIFECYCLE.md`
  - `aidocs/INTERPRETERS.md`

### 3) Current Behavior Anchors

- Ranking currently consumes `Keywords`, `Usage`, `Summary`, `Helptext` (`bot/builtins.go`).
- `Usage` fallback can be derived from first `Helptext` line (`bot/builtins.go`).
- `InputMatcher` still defines `Helptext` in schema (`bot/tasks.go`).

### 4) Proposed Behavior

Changes:

- Remove `Helptext` from schema and ranking/rendering.
- Build deterministic searchable tokens from:
  - plugin name
  - command
  - usage
  - summary
  - explicit keywords (if present)
- Keep exact keyword match high-scoring.

No change:

- Plugin availability filtering logic.
- Group filtering via `usergroups`.
- Hidden command gating.
- Help command interfaces: `help`, `commands`, `help-all`, fallback suggestions.

### 5) Invariant Impact Check

- Startup determinism preserved: Yes.
- Explicit control flow preserved: Yes.
- Shared auth/policy remains in engine flows: Yes.
- Permission checks remain username-based: Yes.
- Connector ordering guarantees preserved: Yes.
- Config precedence still explicit: Yes.
- Multi-connector isolation preserved: Yes.

### 6) Cross-Cutting Concerns

- Config schema migration required (`Helptext` removed).
- Ranking behavior must stay deterministic and regression-tested.
- Docs and examples must be updated in the same change series.

### 7) Concurrency Risks

- No new concurrency surfaces required if ranking remains per-request and read-only.
- If token precomputation is added, compute under config load path and publish atomically with existing config structures.

### 8) Backward Compatibility

- Config-level breaking change is acceptable per `aidocs/V3_COMPATIBILITY_CONTRACT.md`.
- Must update `UPGRADING-v3.md` with explicit migration from `Helptext`.

### 9) Validation Plan

- Unit tests for scoring, metadata collection, validation.
- Integration checks for `help`, `help-all`, `commands`, fallback response shape.

### 10) Documentation Plan

- Update:
  - `UPGRADING-v3.md`
  - `aidocs/PIPELINE_LIFECYCLE.md`
  - `aidocs/INTERPRETERS.md`

### 11) Waiver

- Not waived.

## Slice Plan

Execute in order. Do not skip validation gates.

### Slice 0: Baseline Snapshot

Purpose: capture current behavior before edits.

Tasks:

1. Run targeted tests and store output.
2. Record current top-level help behavior from integration tests.

Suggested commands:

```bash
go test ./bot -run 'Test(ScoreHelpCommandMatch|RankHelpMatches|FirstHelpLineUsageAndSummary|StripHelpAddressPrefix|HiddenSlashBotExample|CommandAllowsHidden)'
go test ./bot -run 'TestValidateYAMLPlugin(RejectsLegacyHelpKey|RejectsLegacyCommandMatchersKey|AcceptsCommandsKey)'
```

Gate:

- Both commands pass.

### Slice 1: Ranking Refactor (No Schema Break Yet)

Purpose: make engine logic independent of `Helptext` before schema removal.

Tasks:

1. In `bot/builtins.go`:
  - Stop using `Helptext` for ranking.
  - Stop using `Helptext` for `Usage`/`Summary` fallback.
  - Add deterministic token expansion helper for optional keywords:
    - baseline tokens from plugin/command/usage/summary.
    - union explicit keywords.
2. Keep score priorities explicit:
  - exact command
  - exact plugin
  - exact keyword/token
  - substring and token overlap fallback
3. Preserve stable sort tiebreaks.

Implementation notes:

- Keep exact keyword matches high (existing semantics around score ~92 are acceptable).
- Do not include generic stopwords in auto token expansion if they reduce quality.
- Avoid regex-heavy runtime work per lookup if simple tokenization is enough.

Tests:

1. Update/add tests in `bot/builtins_help_metadata_test.go`:
  - ranking still finds list commands for natural query phrases.
  - command with no explicit keywords remains discoverable.
  - exact keyword still outranks loose substring.
2. Remove tests that assert `Helptext`-based fallback behavior.

Gate:

```bash
go test ./bot -run 'Test(ScoreHelpCommandMatch|RankHelpMatches|NormalizeFallbackTerm|HelpTokenEquivalent)'
```

Must pass.

### Slice 2: Remove `Helptext` From Schema

Purpose: enforce new metadata contract.

Tasks:

1. Remove `Helptext` field from `InputMatcher` in `bot/tasks.go`.
2. Remove references to matcher helptext in help metadata collection.
3. Add/replace validation test in `bot/taskconf_commands_test.go`:
  - plugin YAML containing `Helptext` under `Commands` should fail validation.
  - error should mention unknown key `Helptext`.

Gate:

```bash
go test ./bot -run 'TestValidateYAMLPlugin'
```

Must pass.

### Slice 3: Migrate Repository Metadata

Purpose: eliminate `Helptext` usage in shipped config and default plugin YAML blobs.

Tasks:

1. Remove `Helptext` entries from all `conf/plugins/*.yaml`.
2. Remove `Helptext` entries from default config blobs in:
  - `plugins/go-lists/lists.go`
  - `plugins/go-links/links.go`
  - `plugins/go-knock/knock.go`
  - `plugins/go-new-robot/new_robot.go`
  - `goplugins/duo/duo.go`
3. Ensure each command still has reasonable `Usage` and `Summary`.
4. Keep `Keywords` only where they add meaningful discoverability.

Gate:

```bash
rg -n 'Helptext:' conf/plugins plugins goplugins
```

Expected: no hits in active command metadata surfaces.

### Slice 4: Docs + Migration

Purpose: keep docs aligned with implementation.

Tasks:

1. Update `UPGRADING-v3.md`:
  - remove `Helptext` recommendation.
  - add migration guidance: delete `Helptext`, keep `Usage`/`Summary`, optional `Keywords`.
2. Update `aidocs/PIPELINE_LIFECYCLE.md`:
  - remove `Helptext` from InputMatcher help metadata list.
3. Update `aidocs/INTERPRETERS.md` examples:
  - no `Helptext` in command examples.

Gate:

```bash
rg -n 'Helptext' UPGRADING-v3.md aidocs/PIPELINE_LIFECYCLE.md aidocs/INTERPRETERS.md
```

Expected: no mentions except explicit migration history notes if intentionally retained.

### Slice 5: Regression and Final Validation

Purpose: ensure user-visible behavior stays solid.

Tasks:

1. Run focused bot tests.
2. Run integration help tests.
3. Spot-check fallback suggestions still produce useful top matches.

Suggested commands:

```bash
go test ./bot
TEST=Help make test
TEST=Devel make test
```

If environment blocks network/listener operations, capture that explicitly in notes.

## Implementation Checklist

Use this mechanical checklist during coding:

1. [ ] `Helptext` no longer appears in `bot/tasks.go` `InputMatcher`.
2. [ ] `bot/builtins.go` has no ranking/render dependency on `Helptext`.
3. [ ] Auto-token path exists so missing `Keywords` is safe.
4. [ ] Exact keyword still ranks high.
5. [ ] No `Helptext:` left in repository-owned plugin command metadata.
6. [ ] Validation tests cover rejection of `Helptext`.
7. [ ] Docs updated in same branch.
8. [ ] All required tests run and results recorded.

## Suggested Commit Slicing

Use small commits to reduce rework risk:

1. `help: decouple ranking from helptext; add tokenized optional-keyword path`
2. `config: remove Helptext from InputMatcher and validation expectations`
3. `config: migrate default plugin command metadata away from Helptext`
4. `docs: update v3 upgrade and aidocs metadata contract`
5. `test: add ranking regression cases for optional keywords`

## Common Pitfalls

1. Accidentally lowering exact keyword score below command/plugin exact matches.
2. Over-aggressive stopword filtering that drops useful terms (`list`, `job`, `log`).
3. Forgetting default-config blobs embedded in Go sources.
4. Updating docs before schema/tests, causing transient mismatch.

## Done Criteria

Proposal 1 is done when:

1. `Helptext` is fully removed from active schema and shipped metadata.
2. `Keywords` can be omitted without poor discoverability.
3. Exact keyword lookup remains high-confidence.
4. Help and fallback remain deterministic, fast, and readable.
5. Migration and architecture docs are consistent with code.

