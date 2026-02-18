# Username-Authoritative Identity Redesign (Design)

## Context

Current v3 identity behavior still carries engine-owned per-protocol `UserMap` mappings and engine-side `username -> internalID` assumptions for outbound messaging and some runtime context paths.

Current bot identity behavior is also still partly global:
- connectors call `SetBotID`, but engine stores a single global bot ID,
- only the primary connector is allowed to set that global bot identity,
- some engine paths still compare incoming `UserID` to that one global bot ID.

That is awkward for multi-protocol operation where:
- security decisions are already username-based,
- connectors are the transport trust boundary,
- protocol-local ID semantics vary significantly by connector.

## Design Decision

Adopt a username-authoritative contract:

1. Engine security identity is canonical username only.
2. Connectors are responsible for transport authentication and canonical username resolution.
3. Engine `UserMap` is removed as a shared contract.
4. Engine-to-connector user-targeting is username-based only.
5. Connector-local identity data is protocol-specific and documented per connector.
6. Bot transport IDs are protocol-scoped in engine runtime state (not a single global ID).
7. `GetBotAttribute("id")` resolves by execution context:
   - plugin/message pipelines: protocol that triggered the pipeline
   - jobs/init/scheduled pipelines: `DefaultProtocol`

`UserID` handling:
- Inbound `UserID` can remain as metadata/provenance.
- Outbound engine contract does not depend on `UserID`.
- Engine policy checks never depend on `UserID`.
- Bot `UserID` exposed via `GetBotAttribute("id")` is context-aware protocol metadata.

## Robot Identity Resolution Rules

- Connectors continue calling `SetBotID(id)` (no connector API signature change).
- Engine records bot ID as `protocol -> botID`.
- Connectors remain the trust boundary for bot/self identification and should set `SelfMessage=true` for bot-authored inbound events.
- Engine routing paths should not depend on one global bot ID comparison.
- For `GetBotAttribute("id")`:
  - if pipeline has inbound protocol context, return that protocol's bot ID,
  - otherwise return the bot ID for runtime `DefaultProtocol`,
  - return empty/unknown when a protocol-scoped bot ID is unavailable.

## Invariant Mapping

Preserved:
- Startup sequence deterministic and explicit.
- Shared auth/business policy remains in engine.
- Permission/policy decisions are protocol-agnostic username-based.
- Connector isolation and per-connector ordering behavior remain.

Redefined:
- Identity mapping source of truth changes from engine-owned `UserMap` to connector-owned protocol mapping.
- `IgnoreUnlistedUsers` gates on trusted connector username + global `UserRoster`.
- Bot ID storage/lookup becomes protocol-scoped instead of global.

## Connector-Specific Consequences

SSH:
- Connector config should support `username -> [pubkeys...]`.
- Connector enforces pubkey authentication and emits canonical username.
- Multiple keys per username are first-class.

Slack:
- Connector-local canonical override map remains necessary for cases where service username and desired robot username mapping differ.
- Existing "engine map wins" semantics move into connector-local config ownership.

Terminal/Test:
- Connector-local user tables continue to define runtime IDs.
- Engine policy remains username-only.

## Memory Model Decision

User-scoped memory binds to username identity across protocols ("parsley is parsley").

Thread-aware memory remains protocol-sensitive when thread IDs are involved.

Compatibility:
- Ephemeral memory compatibility break is acceptable.
- Persistent brain compatibility remains primary.

## Compatibility Priorities

1. Extension API signatures and behavior continuity.
2. Username-based security model continuity.
3. Persistent brain continuity as much as possible.
4. Config compatibility is lower priority; config migration is acceptable.

## Documentation Strategy

Before or during implementation slices:
- Update `aidocs/STARTUP_FLOW.md` identity section to reflect new contract.
- Update connector docs (`aidocs/SSH_CONNECTOR.md`, Slack/Terminal docs as needed) for connector-local identity schemas.
- Keep this design/slice package as migration ledger for reviewers.
