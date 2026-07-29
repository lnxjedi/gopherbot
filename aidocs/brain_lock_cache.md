# Brain Cache and Ownership Decisions

The engine-facing brain is a durable local cache. Remote providers are encrypted
sync backends, not the runtime read path. This keeps normal reads local and
allows writes to commit locally before metered/coalesced cloud sync.

## Compatibility boundary

- `file` is a valid local-only brain.
- Cloud runtime records are v3-only. V2/unversioned import and rollback export
  are explicit CLI operations, not startup branches.
- `pull-brain` chooses remote/legacy data as the source for a local cache.
  `restore-brain` chooses the local cache as the source for remote data.

## Ownership and startup

Each local cache has a persistent random lineage nonce. The remote
`bot:instance-lock` stores a lock ID tied to that lineage and is either `held`
or `released`.

- A held lock may be reclaimed only by the same cache lineage and active lock
  ID. Wall-clock age is not proof of ownership.
- A released lock records the last database version. If it is newer than the
  local cache, startup fails and requires explicit recovery.
- A new cache may hydrate from a released, fully v3 remote brain.
- Startup verifies the last cloud write it believed succeeded. Provider-owned
  retry policy handles eventual consistency; scattered sleeps must not.
- Durable outbox entries are replayed before readiness.
- Ambiguous ownership, stale cache, unreadable metadata, or unresolved
  checkpoint mismatch fails closed with an actionable CLI message.

The startup gate stays closed until brain safety and initial plugin quiescence
complete. Local state is authoritative only after those proofs.

## Shutdown

Stop new work, flush the outbox, then write a persistent `released` lock with
the local database version before shutting down the brain. Do not delete the
lock: its version is the evidence a later cache needs to detect staleness.

The design prevents accidental concurrent robots; it is not multi-writer
conflict resolution.
