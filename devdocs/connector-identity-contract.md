# Connector Identity Contract

This document captures the planned identity contract for v3 multi-protocol cleanup.
It is written for human contributors and reviewers.

Status:
- Proposed and approved in design discussions.
- Implemented through Slice 7 (connector-local identity ownership + username-keyed memory semantics + cleanup).

## Why This Exists

The old model mixed:
- username-based security decisions (admin users, groups, authorization), and
- engine-owned protocol `UserMap` ID mapping.

For multi-protocol behavior, this creates unnecessary coupling. The target model makes security identity explicit and connector-local transport mapping explicit.

## Target Contract Summary

1. Security principal is always `UserName`.
2. Connectors are trusted to authenticate and provide canonical usernames.
3. `UserID` is connector-local metadata/provenance, not a security identity.
4. Engine-to-connector user targeting is username-based (no engine contract on `UserID`).
5. Connector identity configuration is protocol-specific and documented per connector.
6. `IgnoreUnlistedUsers` gates on trusted connector username membership in global `UserRoster`.

## Contract Details

### A) Username Is Authoritative

- All engine policy checks are username-based.
- "Parsley is parsley" across protocols for authz semantics.
- If a connector cannot deterministically map transport identity to canonical username, that connector must reject the message/session.

### B) UserID Is Informational

- Inbound `UserID` may still be present and discoverable.
- Connectors may use `UserID` internally for protocol operations.
- Engine must not use `UserID` as the authorization key.

### C) Connector-Local Mapping Owns Transport Semantics

- Slack may use `internalID <-> username` tables.
- SSH may map `username -> set of allowed pubkeys`.
- Terminal/test may map via connector-local user tables.
- Engine does not define protocol-local identity schema beyond requiring canonical username on inbound.

### D) IgnoreUnlistedUsers Behavior

- If `IgnoreUnlistedUsers` is enabled, the engine allows messages only when:
  - connector provided a canonical username, and
  - that username exists in global `UserRoster`.
- No engine `UserMap` dependency in this gate.

### E) Memory Semantics

- Persistent and ephemeral user-scoped memories should bind to username identity.
- Cross-protocol access to the same username memory is expected.
- Thread-scoped memory remains protocol-aware when thread context is used.

### F) Bot Internal ID Exposure

- Connectors continue to provide bot transport IDs through `SetBotID(...)`.
- Engine stores bot IDs as protocol-scoped runtime state (`protocol -> botID`), not a single global ID.
- `GetBotAttribute("id")` resolves by execution context:
  - plugin/message pipelines: bot ID for the protocol that triggered the pipeline
  - job/init/scheduled pipelines: bot ID for `DefaultProtocol`
- `UserID` remains metadata/provenance and connector-local for transport operations.

## Connector Obligations

Each connector must document:
- identity source of truth (transport auth input),
- canonical username resolution rules,
- ambiguity/failure behavior,
- protocol-local identity config format,
- reload/update behavior for identity maps,
- guarantees for username stability.

## Non-Goals

- Forcing identical identity schemas across connectors.
- Inferring cross-protocol equivalence heuristically.
- Removing protocol-local IDs from connectors themselves.

## Migration Notes (Design-Level)

- Existing robots can update configuration files.
- Persistent brain compatibility is prioritized over config compatibility.
- Ephemeral memory format changes are acceptable if needed.
- Any behavior change to extension API methods must preserve existing signatures and expected semantics.
