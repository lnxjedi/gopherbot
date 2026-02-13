# Upgrading Existing Robots to v3 (Multi-Protocol)

This guide describes user-facing config changes for robots moving from the old single-protocol model to v3 multi-protocol.

## Quick Checklist

1. Set `PrimaryProtocol` in `conf/robot.yaml`.
2. Add `SecondaryProtocols` for additional connectors.
3. Make sure the primary protocol has a valid `ProtocolConfig` source.
4. Move shared policy/config keys (especially `AdminUsers`) to `conf/robot.yaml`.
5. Use global `UserRoster` for directory attributes and per-protocol `UserMap` for connector IDs.
6. Reload and verify runtime with `protocol-list` (or `protocol list`).

## Primary/Secondary Protocol Keys

- Preferred key: `PrimaryProtocol`.
- Legacy alias: `Protocol` (still accepted).
- If both are set and differ, `PrimaryProtocol` wins and a warning is logged.

Example:

```yaml
PrimaryProtocol: slack
SecondaryProtocols: [ "ssh" ]
```

Notes:

- If `SecondaryProtocols` includes the primary protocol, it is ignored with a warning.
- If `SecondaryProtocols` includes `terminal`, it is ignored with a warning (`terminal` is not supported as a secondary protocol).

## Two Supported Config Styles

### 1) Compatibility Style (old include-driven robots)

If your `conf/robot.yaml` still includes connector files (for example `{{ .Include "slack.yaml" }}`), it continues to work.

Behavior:

- Primary protocol config is taken from merged `robot.yaml` `ProtocolConfig`.
- Engine logs a compatibility warning and recommends the new style.

### 2) Recommended Style (new v3 layout)

Use `conf/robot.yaml` for shared/global config and let the engine load per-protocol files directly:

- `conf/<PrimaryProtocol>.yaml` is auto-loaded for primary when `robot.yaml` does not define `ProtocolConfig`.
- each secondary listed in `SecondaryProtocols` is loaded from `conf/<secondary>.yaml`.

In per-protocol files (`conf/<protocol>.yaml`), keep connector-local keys:

- `ProtocolConfig` (required for active connector startup)
- `UserMap` (recommended for all real users)
- optional protocol-local `ChannelRoster`

Do not rely on these keys in `conf/<protocol>.yaml` in recommended style:

- `AdminUsers`
- `BotInfo`
- `Alias`
- `DefaultChannels`
- `DefaultJobChannel`

Those belong in `conf/robot.yaml`.

## Identity Model Changes (`UserRoster` vs `UserMap`)

- `UserRoster` is now the global user directory (username + attributes).
- `UserMap` is per-protocol mapping: `username -> protocol internal user ID`.
- Username rules are strict: lowercase only; uppercase entries are rejected.

`IgnoreUnlistedUsers: true` now requires both:

- user exists in global `UserRoster`
- user exists in that protocol's `UserMap`

### Legacy compatibility bridge

`UserRoster.UserID` is still parsed for backward compatibility, but only as a mapping bridge:

- if `UserMap` is missing an entry, legacy `UserRoster.UserID` can fill it
- if both exist, `UserMap` wins and a warning is logged

For migration, keep attributes in `UserRoster` and move all IDs to `UserMap`.

SSH example (recommended):

```yaml
UserMap:
  parsley: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA..."

UserRoster:
- UserName: parsley
  Email: parsley@example.com
```

## Admin Users Are Global

`AdminUsers` should be defined in `conf/robot.yaml`, not in protocol files.

```yaml
AdminUsers: [ "david" ]
```

Reason: authorization policy is shared across protocols.

## Reload and Runtime Semantics

- Primary connector startup failure is fatal.
- Secondary connector startup failure is logged and does not stop the robot.
- On reload:
  - removed secondaries are stopped
  - configured secondaries are retried
  - primary protocol change is rejected (logged and ignored)

Secondary retries happen when:

- config reload occurs
- admin starts/restarts protocol explicitly

## Primary Protocol Gotcha When Switching

If you switch primary protocol, the new primary must have `ProtocolConfig` available via one of:

- compatibility style: merged `robot.yaml` contains `ProtocolConfig`
- recommended style: `conf/<primary>.yaml` exists and contains `ProtocolConfig`

If neither is true, startup/reload fails for the primary connector.

## Primary-Protocol Admin Commands

These commands are available from the primary protocol:

- `protocol-list` or `protocol list`
- `protocol-start <name>` or `protocol start <name>`
- `protocol-stop <name>` or `protocol stop <name>`
- `protocol-restart <name>` or `protocol restart <name>`
