# Impact Surface Report

## 1) Change Summary

- Slice name: `thread-subscription-protocol-slice6`
- Goal:
  - protocol-scope thread subscription keys to prevent cross-protocol subscription collisions
  - preserve backward compatibility for persisted subscriptions created before protocol was part of the key
  - restore integration-test compatibility for legacy `UserRoster.UserID` configs while keeping directory/runtime identity split
- Out of scope:
  - subscription policy redesign
  - prompt/reply matcher redesign
  - connector behavior changes

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
  - `bot/subscribe_thread.go`
  - `bot/dispatch.go`
  - `bot/conf.go`
  - `bot/subscribe_thread_test.go` (new)
  - `aidocs/TESTING_CURRENT.md`
  - `aidocs/multi-protocol/2026-02-13-thread-subscription-protocol-slice6/*`
- Key functions/types/symbols:
  - `subscriptionMatcher`
  - `tSubs.MarshalJSON()`, `(*tSubs).UnmarshalJSON()`
  - `Robot.Subscribe()`, `Robot.Unsubscribe()`, `expireSubscriptions()`
  - dispatch subscription routing in `bot/dispatch.go`
  - `ConfigLoader` user roster loader type

## 3) Current Behavior Anchors

- Subscription matching currently keys by `(channel, thread)` only.
- In multi-protocol operation this can collide when different connectors share channel/thread names.
- Subscription expiration logging lacks protocol context due missing protocol in key.
- Integration tests currently fail on legacy configs because strict YAML decode rejects `UserRoster.UserID` in global roster entries.

## 4) Proposed Behavior

- Thread subscriptions are keyed by `(protocol, channel, thread)`.
- Persisted subscription JSON supports:
  - new key format with protocol
  - legacy key format without protocol (loaded as protocol-unknown compatibility entries)
- Dispatch first checks protocol-scoped key, then legacy compatibility key on restore-era data.
- Expiration logs include protocol when known.
- Config loader accepts legacy `UserRoster.UserID` fields for compatibility parse, while runtime directory model remains protocol-agnostic.

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes
- Multi-connector isolation preserved?: yes (improved for thread subscriptions)

No invariant is intentionally redefined.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - none.
- Config loading/merge/precedence impact:
  - no precedence changes; compatibility field-acceptance only.
- Execution ordering impact:
  - none; lookup key includes protocol dimension.
- Resource lifecycle impact:
  - none.

## 7) Concurrency Risks

- Shared state touched:
  - `subscriptions.m` guarded by `subscriptions.Lock()`
- Locking assumptions:
  - keep existing lock discipline; no new lock classes.
- Risk level:
  - low.

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - legacy `UserRoster.UserID` configs continue to parse.
  - old persisted subscriptions continue to restore.
- Behavior changes for operators/users:
  - cross-protocol subscription collision risk removed.
  - logs include protocol for subscription expiration when available.
- Migration/fallback plan:
  - no required operator migration for this slice.

## 9) Validation Plan

- Focused tests:
  - subscription marshal/unmarshal with new and old key formats
  - protocol-scoped subscription lookup behavior
  - config loader acceptance of legacy roster `UserID`
- Broader regression tests:
  - `go test ./bot`
  - `go test ./...`
  - `make test`

## 10) Documentation Plan

- `aidocs/TESTING_CURRENT.md` updates:
  - where to inspect integration failure logs (`/tmp/bottest.log`) and quick triage flow.
- Slice docs:
  - add compatibility note and invariants checklist.

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
