# Gopherbot Local Brain Cache Architecture

## Problem

Cloud-backed memories are too slow and too expensive to treat as the normal
runtime read path. Cloudflare KV is the most sensitive case: direct reads during
startup and normal command handling add noticeable latency and can consume daily
free-tier operations quickly.

## Runtime model

The v3 engine uses a persistent local file-key cache as the engine-facing brain.
This cache stores encrypted memory payloads plus v3 metadata and a durable
outbox. The normal `Retrieve`, `List`, `Store`, and `Delete` calls do not read
from cloud providers.

`Brain: file` is local-only cache mode. It is valid for development robots and
does not warn merely because no remote backend exists.

Cloud providers are v3 remote sync backends:

- Cloudflare KV
- AWS DynamoDB
- Google Firestore

All shipped cloud providers expose the same v3 remote contract: identity,
metadata listing, point reads, puts, tombstone deletes, sync policy, and
shutdown.

## Local cache layout

`BrainCache.Directory` contains:

- `control.json`: cache format, completion state, local version counter, cache
  nonce/lock state, and remote checkpoint.
- `data/`: encrypted payload blobs keyed by encoded memory key.
- `meta/`: v3 metadata per memory key.
- `outbox/`: durable pending cloud writes keyed by memory key.
- `write-budget.json`: persisted per-day cloud write counter when the selected
  provider sets a write budget.

The cache uses atomic file replacement for control, metadata, outbox, and
payload writes. This keeps the implementation simple and avoids an embedded
database dependency for expected ChatOps memory sizes.

## Startup paths

Normal startup has three paths:

1. **Fast restart:** a complete local cache exists. Startup verifies the local
   checkpoint with one remote point-read and does not scan the cloud.
2. **Fresh VM with v3 remote:** no local cache exists. Startup scans v3 remote
   metadata, downloads v3 records, writes the local cache, and continues.
3. **Fresh VM with v2/unversioned remote:** startup exits cleanly and tells the
   owner to run `gopherbot pull-brain`.

The v3 runtime requires a v3 remote brain for cloud-backed robots. It may detect
v2/unversioned records so it can refuse startup, but it must not import or
upgrade v2 records during normal robot startup.

## Sync behavior

Local writes commit before returning to the engine. For remote brains, the cache
then writes or replaces an outbox entry and wakes the sync worker.

The sync worker:

- coalesces rapid updates to the same memory key
- applies provider `MinWriteInterval`
- enforces provider `WriteBudgetPerDay` using `write-budget.json`
- leaves unsynced work in `outbox/` when the remote write fails or the daily
  budget is exhausted
- coalesces repeated updates to the same memory key by replacing that key's
  outbox entry, so only the latest version is sent if the earlier version has
  not reached the cloud yet
- flushes until clean during shutdown/restart; `FlushOnShutdownMaxDuration`
  controls periodic warning cadence, not a hard cutoff

The engine's instance-lock memory key is synced immediately because it protects
single-robot ownership.

## CLI migration paths

V2 compatibility is CLI-only:

- `gopherbot pull-brain` imports v2/v3 remote records or legacy file-brain data
  into the local v3 cache. It excludes the cloud `bot:instance-lock`, which can
  be inspected directly with `fetch -cloud bot:instance-lock`, and does not
  modify cloud by default.
- `gopherbot pull-brain -upgrade-cloud-v3` additionally writes upgraded v3
  records to cloud.
- `gopherbot restore-brain` writes the local cache to the configured cloud
  provider in v3 format.
- `gopherbot restore-brain -v2` writes v2-compatible cloud records for rollback
  to v2 code.
- `restore-brain -force` removes remote keys absent from the local cache.

The local cache does not store or enforce the cloud driver identity. If two
configured remote brains are in sync with the cache's database version and
checkpoint, the same `BrainCache.Directory` can be used with either backend.

This keeps normal startup deterministic and avoids hidden one-way upgrades.

Existing memory commands are cache-first with explicit cloud inspection:

- `fetch <key>` and `list` read the local cache only.
- `fetch -validate-cloud <key>` verifies the local cached record against the v3
  cloud record before printing the local value.
- `fetch -cloud <key>` reads directly from v3 cloud; `-update-cache` can repair
  an existing complete local cache for that key.
- `list -cloud` lists cloud keys.
- `store <key>` and `delete <key>` write the local cache and flush the cloud
  operation before reporting success.
- `flush-brain` drains queued local outbox work.

Whenever a CLI command intentionally touches the cloud, it writes cache-sync
status to stderr so stdout remains scriptable command output.

## Configuration

Engine-owned cache settings:

```yaml
BrainCache:
  Directory: state/brain-cache
```

Provider credentials and provider-sensitive sync tuning stay in
`conf/brains/<Brain>.yaml`. The cache asks the remote backend for sync policy
instead of duplicating provider-specific settings in `BrainCache`.

The v3 engine always repairs or verifies cloud-backed cache state before
command readiness. Pending local outbox entries are replayed during startup, and
there is no configurable mode that allows command handling while cloud
persistence is known to be dirty.
