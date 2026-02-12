# Upgrading Existing Robots to v3 (Multi-Protocol)

This guide covers config changes needed when moving an existing robot to the v3 multi-protocol model.

## Summary of What Changed

- `PrimaryProtocol` is now the preferred key in `conf/robot.yaml`.
- `Protocol` is still accepted as a legacy alias.
- `SecondaryProtocols` enables additional connectors to run at the same time.
- Secondary connector roster/config is loaded from `conf/<protocol>.yaml` (for example `conf/ssh.yaml`, `conf/slack.yaml`).

## Safe Migration Pattern

1. Keep your current protocol as primary.
2. Add `SecondaryProtocols` one protocol at a time.
3. Add `conf/<protocol>.yaml` for each secondary.
4. Reload and verify with `protocol-list`.
5. Only then consider switching primary protocol.

## Required Config Checks

## 1) Primary protocol selection

In `conf/robot.yaml`, use:

```yaml
PrimaryProtocol: slack
```

`Protocol` still works, but `PrimaryProtocol` should be used going forward.

If both are set and differ, `PrimaryProtocol` wins and a warning is logged.

## 2) Secondary list

Example:

```yaml
SecondaryProtocols: [ "ssh" ]
```

If `SecondaryProtocols` includes the current primary protocol, it is ignored with a warning.
If `SecondaryProtocols` includes `terminal`, it is ignored with a warning (terminal is not supported as a secondary protocol).

## 3) Per-protocol config files

For each secondary, create a matching file:

- `SecondaryProtocols: [ "ssh" ]` -> `conf/ssh.yaml`
- `SecondaryProtocols: [ "slack" ]` -> `conf/slack.yaml`

Include at least:

- `UserRoster`
- `ProtocolConfig`

Do not rely on `AdminUsers` inside protocol files for v3 multi-protocol behavior.

## 4) Admin users are global (robot.yaml)

`AdminUsers` should be defined in `conf/robot.yaml`, not in `conf/<protocol>.yaml`.

Example:

```yaml
# conf/robot.yaml
AdminUsers: [ "david" ]
```

Reason: admin authorization is shared engine policy and applies across protocols.

## 5) Primary protocol gotcha when switching

If you switch the primary protocol, make sure your merged `robot.yaml` still provides the primary protocol's `ProtocolConfig`.

Common failure mode:

- old `robot.yaml` only includes `terminal.yaml`/`slack.yaml` conditionally
- primary changed to `ssh`
- no `ProtocolConfig` provided for ssh in merged config
- connector fails on startup.

## 6) Username rules

Usernames in rosters must be lowercase. Uppercase names are rejected.

For SSH connector, `UserRoster.UserID` must be the normalized public key line (`<type> <base64>`), for example:

```yaml
UserRoster:
- UserName: parsley
  UserID: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA..."
```

## Runtime Behavior Notes

- Primary connector failure at startup is fatal.
- Secondary connector failure is logged and does not stop the robot.
- On reload:
  - removed secondaries are stopped
  - configured secondaries are retried
  - primary protocol change is rejected (logged and ignored).
- Slack connector lifecycle state is connector-scoped in v3 multi-protocol work, so
  `protocol-stop slack`, `protocol-start slack`, `protocol-restart slack`, and
  secondary remove/add on reload are expected to work without relying on process restart.

## Admin Commands (Primary Protocol Only)

- `protocol-list` / `protocol list`
- `protocol-start <name>` / `protocol start <name>`
- `protocol-stop <name>` / `protocol stop <name>`
- `protocol-restart <name>` / `protocol restart <name>`
