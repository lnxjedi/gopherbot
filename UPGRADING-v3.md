# Upgrading Existing Robots to v3 (Multi-Protocol)

This guide describes user-facing config changes for robots moving from the old single-protocol model to v3 multi-protocol.

Compatibility scope for this guide:
- Existing plugin/job/task scripts are expected to keep working across v2 -> v3.
- Configuration is expected to require migration as v3 architecture evolves.

## Quick Checklist

1. Set `PrimaryProtocol` in `conf/robot.yaml`.
2. Add `SecondaryProtocols` for additional connectors.
3. Make sure the primary protocol has a valid `ProtocolConfig` source.
4. Move shared policy/config keys (especially `AdminUsers`) to `conf/robot.yaml`.
5. Use global `UserRoster` for canonical usernames and directory attributes.
6. Ensure connector-emitted usernames match global `UserRoster` entries (especially with `IgnoreUnlistedUsers: true`).
7. Confirm connector-specific identity mapping config is correct inside each connector `ProtocolConfig` (for example Slack `ProtocolConfig.UserMap`, SSH `ProtocolConfig.UserKeys`).
8. Reload and verify runtime with `protocol-list` (or `protocol list`).

## 2026-02-18 Provider Config Layout Update (Slice 1)

Provider-specific configuration moved out of `conf/robot.yaml`:

- brain provider config now lives in `conf/brains/<Brain>.yaml` under top-level `BrainConfig`
- history provider config now lives in `conf/history/<HistoryProvider>.yaml` under top-level `HistoryConfig`

Breaking config change:

- top-level `BrainConfig` and `HistoryConfig` keys in `conf/robot.yaml` are invalid and fail config load

Upgrade actions:

1. Keep only provider selectors in `conf/robot.yaml` (for example `Brain`, `HistoryProvider`).
2. Move old `BrainConfig` block to `conf/brains/<provider>.yaml`.
3. Move old `HistoryConfig` block to `conf/history/<provider>.yaml`.
4. Verify selected provider files exist and contain the required top-level key (`BrainConfig` or `HistoryConfig`).

## 2026-02-18 Username Identity Update (Slices 1 + 2 + 3 + 3b)

These slices changed runtime behavior in ways that matter for upgrades:

- Outbound engine-to-connector user sends are now username-based.
- `IgnoreUnlistedUsers` now gates on trusted connector username membership in global `UserRoster`.
- Inbound `UserID` remains metadata/provenance, but is no longer required for engine policy checks.
- Engine no longer owns/distributes per-protocol `UserMap`; mapping is connector-local inside `ProtocolConfig`.
- Bot internal IDs are protocol-scoped in engine runtime state (`protocol -> botID`).
- `GetBotAttribute("id")` now resolves by context:
  - plugins/messages: triggering protocol bot ID
  - jobs/init/scheduled: `DefaultProtocol` bot ID

Upgrade actions:

1. Verify each connector emits canonical usernames that match `UserRoster.UserName`.
2. If `IgnoreUnlistedUsers: true`, ensure each allowed user exists in global `UserRoster`.
3. Validate user-targeted replies/DMs by username in each active connector.

Slack-specific notes:

1. Move any top-level Slack `UserMap` entries into `ProtocolConfig.UserMap`.
2. Slack connector now treats `ProtocolConfig.UserMap` as canonical username-to-ID mapping.
3. Top-level `UserMap` keys are invalid and now fail config load.

Terminal/Test-specific notes:

1. Connector-local user IDs come from each connector's `ProtocolConfig.Users` entries.
2. Do not use legacy top-level `UserMap`/`AppendUserMap` keys in terminal/test protocol config files.

Global note:
- Top-level `UserMap` in `conf/robot.yaml` is also invalid and fails config load.

Memory keying note:
- Ephemeral user-scoped memory is now keyed by canonical username (not connector `UserID`).
- Thread-scoped ephemeral memory now includes protocol context with thread ID.
- Existing persisted ephemeral-memory entries keyed by old `UserID` semantics may not be recalled after upgrade.
- `GetBotAttribute("id")` is runtime protocol-scoped and no longer falls back to legacy global `BotInfo.UserID` state.

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
- optional protocol-local `ChannelRoster`

Connector identity mapping now lives inside `ProtocolConfig`, for example:
- Slack: `ProtocolConfig.UserMap`
- SSH: `ProtocolConfig.UserKeys` as a list of `{UserName, PublicKeys}` entries (supports multiple keys per username)

Do not rely on these keys in `conf/<protocol>.yaml` in recommended style:

- `AdminUsers`
- `BotInfo`
- `Alias`
- `DefaultChannels`
- `DefaultJobChannel`

Those belong in `conf/robot.yaml`.

Validation note:
- Top-level `UserMap` is invalid in both `conf/robot.yaml` and `conf/<protocol>.yaml`.
- Move mapping data into connector-specific `ProtocolConfig` fields.

## Identity Model Changes (`UserRoster` and Connector-Local Mapping)

- `UserRoster` is now the global user directory (canonical username + attributes).
- Username rules are strict: lowercase only; uppercase entries are rejected.
- Connector-local identity mapping format is protocol-specific and configured in each connector's `ProtocolConfig`.

`IgnoreUnlistedUsers: true` requires:

- connector-authenticated canonical username exists in global `UserRoster`

Notes:

- Engine policy decisions are username-based.

SSH example (recommended):

```yaml
ProtocolConfig:
  UserKeys:
  - UserName: parsley
    PublicKeys:
    - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...key1"
    - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...key2"

UserRoster:
- UserName: parsley
  Email: parsley@example.com
```

If you want to intentionally start with no SSH users, set:

```yaml
ProtocolConfig:
  UserKeys: []
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

## Plugin Command/Help Metadata Migration

v3 help/discovery uses command-linked metadata under each command matcher.

- Directed-command key in plugin config: `Commands`.
- Legacy directed-command key `CommandMatchers` is no longer supported.
- Legacy top-level plugin `Help` is no longer supported.

Recommended command entry fields:

- `Command`, `Regex`
- `Usage`, `Summary`
- optional `Examples`, `Keywords`

Removed field:

- `Helptext` is no longer supported in command entries and now fails validation.

Field semantics and authoring conventions:

- `Usage` is command-body syntax only (no bot name or alias prefix).
  - Good: `Usage: "list lists"`
  - Avoid: `Usage: "(alias) list lists"` or `Usage: "(bot), list lists"`
- `Examples` should use placeholders, not hardcoded names/aliases:
  - Use `(alias)` for concise CLI-like commands (for example `(alias) reboot-server Omega`).
  - Use `(bot)` for conversational commands (for example `(bot) tell me a joke`).
- Hidden-capable command examples:
  - When a command is listed in plugin `AllowedHiddenCommands`, built-in help may render `(bot)` examples as slash-addressed forms (for example `/(bot) whoami` rendered as `/Clu whoami`).
- `Keywords` are optional and used for explicit help/fallback relevance boosts.
- Help search automatically indexes command metadata (`plugin`, `command`, `usage`, `summary`) even when `Keywords` are omitted.

## Built-in Help and Fallback Behavior

Built-in help commands are now metadata-driven:

- `(alias) help <keyword>`: ranked command search
- `(alias) commands`: command groups available in current channel
- `(alias) help-all`: detailed command list including global commands

Built-in unmatched-command fallback now returns algorithmic closest matches using the same command metadata and ranking logic.

## Authorizer `usergroups` Contract (Help Filtering)

For group-aware help visibility, authorizer plugins can implement an optional:

- `usergroups <username> <result_parameter>`

Expected behavior:

- return `Success` and set `result_parameter` via `SetParameter(...)` to a JSON array of group names, e.g. `["Helpdesk","Ops"]`
- return `NotFound` when group membership is intentionally unknown/indeterminate (for example, slow external policy checks)
- errors are treated the same as indeterminate membership for help filtering

Help/fallback filtering behavior:

- if `usergroups` returns usable groups, commands requiring auth are filtered by `AuthRequire`
- if `usergroups` is not implemented, returns `NotFound`, or errors, help output is not group-filtered (no-filter fallback)

## Hidden Command Addressing

Hidden command execution now requires both:

- command is listed in plugin `AllowedHiddenCommands`
- hidden message is robot-addressed:
  - connector-routed bot message (`BotMessage=true`, e.g. Slack slash command), or
  - name-addressed hidden message (`/<botname> <command>` in connectors like SSH)

Practical migration note:
- plain hidden `/<command>` is not treated as a robot-addressed hidden command by default.
