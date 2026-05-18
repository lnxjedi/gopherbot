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
6. Ensure connector-emitted validated usernames match global `UserRoster` entries (especially with `IgnoreUnlistedUsers: true`).
7. Confirm connector-specific identity mapping config is correct inside each connector `ProtocolConfig` (for example Slack `ProtocolConfig.UserMap`, SSH `ProtocolConfig.UserKeys`).
8. Confirm your preferred `DefaultMessageFormat` (v3 default is now `BasicMarkdown`; set `Raw` explicitly to preserve legacy protocol-native output).
9. Search custom config for `{{ decrypt` and move encrypted values into
   custom-only `conf/variables/*.yaml` `Secrets`.
10. Move environment-specific plaintext deployment values into
    `conf/variables/*.yaml` `Variables` and reference them with
    `{{ variable "NAME" }}`.
11. Review `DefaultJobChannel` and any `TimeOuts` values so pipeline alerts
    reach the right operator channel.
12. Reload and verify runtime with `protocol-list` (or `protocol list`).

## 2.9.0 Pre-v3 Pilot Checklist

For non-critical pilot robots before tagging 2.9.0:

1. Run `gopherbot validate <robot-repo>` or a clean startup with the intended
   `GOPHER_ENVIRONMENT`.
2. Confirm no custom config still contains `{{ decrypt`.
3. Verify the selected environment loads the expected
   `conf/variables/common.yaml` and `conf/variables/<environment>.yaml`
   values.
4. Exercise the primary connector plus any secondary connectors used by the
   robot.
5. Check canonical username mapping with one admin and one non-admin user,
   especially when `IgnoreUnlistedUsers: true`.
6. Exercise hidden/admin commands that operators rely on, including `ps`,
   `ps -v`, and `get-pipeline-log <wid>`.
7. Trigger one harmless failing test command or pilot-only pipeline to confirm
   failure alerts and recent log excerpts reach the expected operator channel.
8. If the robot uses setuid privilege separation, perform the manual
   host-level privsep validation from `AGENTS.md` / the execution security docs
   before treating that deployment as representative.

During the 2.9.0 pilot window, prefer bugfixes and UX fixes only. Avoid new
configuration-breaking changes unless a pilot uncovers a critical defect.

## 2026-05-06 Environment-Scoped Secrets And Variables

Inline config-template decryption was removed for v3.

Breaking config change:

- `{{ decrypt "..." }}` is no longer valid in any config template.
- Remaining uses fail startup or `gopherbot validate` with a migration hint.
- Encrypted values now belong under custom robot `conf/variables/*.yaml`
  `Secrets` and are referenced with `{{ secret "NAME" }}`.

Variables files are custom-only:

```yaml
# conf/variables/common.yaml
Secrets:
  WEATHER_API_KEY: "<ciphertext from gopherbot encrypt>"
Variables:
  OUTPUT_CHANNEL: "jobs"
```

```yaml
# conf/variables/development.yaml
Secrets:
  WEATHER_API_KEY: "<development ciphertext>"
Variables:
  OUTPUT_CHANNEL: "dev-jobs"
```

Use them from config:

```yaml
ParameterSets:
  weather:
    Parameters:
    - Name: WEATHER_API_KEY
      Value: {{ secret "WEATHER_API_KEY" | printf "%q" }}

DefaultJobChannel: {{ variable "OUTPUT_CHANNEL" | printf "%q" }}
```

Upgrade actions:

1. Search custom config for `{{ decrypt`.
2. Move each ciphertext into `conf/variables/common.yaml` or the appropriate
   `conf/variables/<GOPHER_ENVIRONMENT>.yaml` under `Secrets`.
3. Replace inline decrypt calls with `{{ secret "NAME" }}`.
4. Put plaintext deployment values that vary by environment under `Variables`
   and reference them with `{{ variable "NAME" }}`.
5. For separate environment key domains, generate or install
   `binary-encrypted-key.<environment>` and encrypt that environment's secrets
   with the matching active data key.

New helper:

```bash
gopherbot genkey -environment development -write
```

`genkey` creates a fresh encrypted binary key using the current
`GOPHER_ENCRYPTION_KEY`. It writes `binary-encrypted-key` for production and
`binary-encrypted-key.<environment>` for non-production environments.

## 2026-05-18 RaisePriv API Removed

`RaisePriv` has been removed from the Go/Yaegi Robot API and from provider
handler interfaces.

The old method was tied to thread-scoped privilege switching. Gopherbot v3
privilege separation is process-scoped instead:

- the parent engine and compiled-in Go extensions run as the invoking robot user
- file-backed extensions commit once in a child process before extension code starts
- unprivileged children cannot switch back to the invoking user

Custom Go or Yaegi extensions that call `RaisePriv` must remove those calls. If
an operation needs invoking-user file or network authority, run it in a
privileged file-backed extension or trusted compiled-in code. If it needs
unprivileged `nobody` authority, it must run as an unprivileged file-backed
extension child.

## 2026-04-28 Privsep Child Process And Supplementary Groups

Privilege separation for file-backed extensions now uses one-shot child processes. The parent engine selects a child role, and `pipeline-child-exec` / `pipeline-child-rpc` commit to that role before running external scripts or built-in interpreters.

New root `robot.yaml` config surface:

```yaml
PrivsepAllowAllSupplementaryGroups: false
PrivsepAllowedSupplementaryGroups: []
```

Installed `conf/robot.yaml` defaults to fail-closed supplementary-group policy. Existing setuid privsep deployments must verify their setuid/setgid install and retained groups before upgrading.

Upgrade actions:

1. Install the binary with both setuid and setgid bits for the unprivileged account, normally `nobody:nobody`.
2. Prefer granting privileged host access directly to the invoking robot user, not through broad groups such as `%wheel` or `%admin`.
3. If the platform retains supplementary groups for unprivileged children, either list the numeric group IDs in `PrivsepAllowedSupplementaryGroups` or explicitly set `PrivsepAllowAllSupplementaryGroups: true` after accepting the weaker boundary.
4. On Linux EC2 deployments, consider UID-scoped firewall rules blocking the unprivileged UID from instance metadata endpoints (`169.254.169.254` and `[fd00:ec2::254]` when IPv6 IMDS is enabled).

## 2026-02-20 BasicMarkdown Default Format Update

`DefaultMessageFormat` now defaults to `BasicMarkdown` for v3.

Behavior change:

- If `DefaultMessageFormat` is omitted, outgoing messages now use `BasicMarkdown`.
- Existing robots that rely on connector-native `Raw` behavior should set:

```yaml
DefaultMessageFormat: Raw
```

Notes:

- `Raw`, `Fixed`, and `Variable` remain supported.
- `BasicMarkdown` is additive and does not renumber existing format values.

## 2026-03-27 Credentialed Shipped Extensions Activation Rule

Credentialed extensions shipped with the engine are no longer assumed active in the default robot.

Behavior and config guidance:

- Extensions that need owner-supplied API credentials, OAuth client secrets, or similar secret material should be enabled explicitly in custom robot config only after the owner provides those credentials.
- For user-linked identity providers, any plugin/job/task that calls `GetIdentityCredential`, `LinkOAuth2Identity`, or `UnlinkIdentity` must have the provider's `CredentialParameterSet` attached in its own `ParameterSets`.
- If that attachment is missing, the identity methods return `IdentityConfigError` and the engine logs the missing attachment details for operators.

Upgrade actions:

1. Move credentialed opt-in extensions such as `github-link` into custom robot config if you want to use them.
2. Create the required `ParameterSets` in custom config and attach them explicitly to each extension/job/task that uses that provider.
3. Do not rely on stock defaults to activate credentialed extensions automatically.

## 2026-04-19 Google Chat Variable Portability Note

Google Chat `Variable` output now uses connector-local homoglyph substitution for punctuation that Google Chat otherwise reparses as text-message formatting.

Behavior note:

- This improves approximate visual literal display for Google Chat `Variable` sends.
- It does not preserve byte-for-byte authored text, so copy/paste fidelity is intentionally weaker than Slack's block-backed `Variable` rendering.
- Google Chat `Raw` remains protocol-native passthrough and is therefore also non-portable for literal display.

Guidance:

1. Prefer `BasicMarkdown` for portable rich formatting across SaaS chat connectors.
2. Prefer `Fixed` when you need stable literal-ish display in both Slack and Google Chat.
3. Treat `Variable` and `Raw` as connector-sensitive modes on SaaS chat connectors rather than portable formatting contracts.

Forward-looking note:

- For a future v4 planning pass, `Variable` and possibly `Raw` should be reviewed as candidates for de-emphasis or deprecation as portable formats on SaaS chat connectors.

## 2026-04-19 Google Chat SelfID Config Note

Google Chat now supports a connector-local `ProtocolConfig.SelfID` value for the bot's numeric `users/{id}`.

Behavior and guidance:

- Do not place the robot's own numeric Google Chat ID in `ProtocolConfig.UserMap`.
- Use `ProtocolConfig.UserMap` only for human canonical username mapping.
- Use `ProtocolConfig.SelfID` for the bot's own numeric Google Chat identity when available.
- Administrators can bootstrap this value with the connector-owned `google validate robot` command from any validated admin DM or hidden context, then persist the learned ID into custom Google Chat protocol config if desired.

Why this exists:

- Google Chat can return the bot's own messages and mention annotations with a numeric bot `users/{id}` instead of the alias `users/app`.
- Separating `SelfID` from `UserMap` lets the connector recognize self messages and bot mentions without forcing the robot to masquerade as a human roster mapping.

## 2026-04-23 Robot Administration Improvements

This release adds pipeline timeout monitoring, richer operator-facing crash visibility, and a broader hidden-command admin surface.

New config surface:

```yaml
TimeOuts:
  Plugin:
    Warn: 7m
    Kill: 14m
  Job:
    Warn: 1h
    Kill: 2h
```

Per-plugin/per-job overrides now live in custom task config:

```yaml
TimeOuts:
  Warn: 15m
  Kill: 30m
```

Upgrade notes:

1. `TimeOuts.Plugin.*` and `TimeOuts.Job.*` are global defaults in `conf/robot.yaml`.
2. `TimeOuts.Warn` / `TimeOuts.Kill` in `conf/plugins/<name>.yaml` or `conf/jobs/<name>.yaml` override those defaults for that task.
3. Explicit `0` disables that threshold for the task.
4. When both are non-zero, `Kill` must be greater than `Warn`.
5. Timeout kill is only enforced for killable child work (external executable or RPC-backed child pipelines). Compiled-in Go work produces warn/manual-intervention alerts but is not force-killed.

Operator workflow changes:

- `ps` now defaults to WID/PWID/type/start/age/task view and hides PID.
- `ps -v` includes PID and execution class details.
- `get-pipeline-log <wid>` shows the live in-memory log buffer for an active pipeline; `wid-log <wid>` is accepted as a shorter synonym.
- Crash/timeout alerts now prefer operator/job-channel notifications with recent log excerpts instead of relying only on `<plugin>-fail.log`.

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
- `IgnoreUnlistedUsers` now gates on trusted connector username membership in global `UserRoster`, where trust means the connector emitted `ValidatedUser=true` for that `UserID -> UserName` mapping.
- Inbound `UserID` remains metadata/provenance, but is no longer required for engine policy checks.
- Engine no longer owns/distributes per-protocol `UserMap`; mapping is connector-local inside `ProtocolConfig`.
- Bot internal IDs are protocol-scoped in engine runtime state (`protocol -> botID`).
- `GetBotAttribute("id")` now resolves by context:
  - plugins/messages: triggering protocol bot ID
  - jobs/init/scheduled: `DefaultProtocol` bot ID

Upgrade actions:

1. Verify each connector emits validated canonical usernames that match `UserRoster.UserName`.
2. If `IgnoreUnlistedUsers: true`, ensure each allowed user exists in global `UserRoster` and is validated by the relevant connector mapping/authentication path.
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

Bot identity for local connectors now lives in shared `BotInfo` in
`conf/robot.yaml`, not protocol-local fields:
- SSH derives its hidden-command bot name from `BotInfo.UserName`
- terminal derives its synthetic bot user and hidden-command bot name from `BotInfo`
- test connector derives bot display/full name from `BotInfo`

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
- the connector marks that inbound mapping as validated (`ValidatedUser=true`)

Notes:

- Engine policy decisions are username-based.
- If `IgnoreUnlistedUsers: false`, inbound traffic for a directory username is still rejected when the connector supplied that username without validating it.
- Administrators can use `validate user <username>` from DM or hidden context to issue a short-lived 7-digit code and learn a user's protocol-local internal ID without weakening the normal inbound trust model.

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
- Private-capable command examples:
  - When a command is listed in plugin `AllowedPrivateCommands`, built-in help may render `(bot)` examples as slash-addressed forms (for example `/(bot) whoami` rendered as `/Clu whoami`).
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

## Private Command Addressing

Private command execution now requires both:

- command is listed in plugin `AllowedPrivateCommands`, `RequiredPrivateCommands`, or covered by `RequireAllCommandsPrivate`
- for hidden/ephemeral invocations, the message is robot-addressed:
  - connector-routed bot message (`BotMessage=true`, e.g. Slack slash command), or
  - name-addressed hidden message (`/<botname> <command>` in connectors like SSH)

Practical migration note:
- plain hidden `/<command>` is not treated as a robot-addressed private command by default.

Built-in private-capable surface is also broader now:

- `builtin-admin` exposes selected admin commands as private-capable through explicit command lists.
- `builtin-history` and `builtin-jobcmd` can also mark specific commands as private-capable through `AllowedPrivateCommands`.
- Private-capable admin/history/job commands still run through the same engine-owned connector support checks and normal admin/authorization/elevation policy.
