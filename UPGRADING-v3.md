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
12. Review brain storage migration. Cloud-backed robots must have a v3 remote
    brain before the v3 engine starts; use `pull-brain` and `restore-brain`
    for v2 import/export.
13. Reload and verify runtime with `protocol-list` (or `protocol list`).

## 2026-05-24 Brain Cache and Cloud Brain Migration

Gopherbot v3 uses an engine-owned local brain cache. The cache is the normal
runtime brain for reads, lists, stores, and deletes. Cloud providers are now
remote sync backends, so steady-state command execution does not read memories
from Cloudflare KV, DynamoDB, or Firestore.

This is a deliberate v3 compatibility boundary:

- The v3 runtime requires v3-formatted cloud brain records for cloud-backed
  robots.
- V2 cloud compatibility exists only in CLI commands.
- A first start on a new VM succeeds when the configured cloud brain is already
  v3-compatible; the engine hydrates the local cache and continues startup.
- A first start with no local cache and a v2/unversioned cloud brain exits
  cleanly and tells the operator to run `gopherbot pull-brain`.
- `Brain: file` remains valid for development robots. It is now local-cache
  only and does not warn or error just because no cloud backend is configured.

### New configuration

Installed defaults now include:

```yaml
BrainCache:
  Directory: {{ env "GOPHER_BRAIN_CACHE_DIRECTORY" | default "<state>/brain-cache" }}
```

`<state>` follows `GOPHER_STATE_DIRECTORY` and defaults to `state`.
`GOPHER_BRAIN_CACHE_DIRECTORY` overrides the cache location directly.

Configuration responsibilities are split:

- `Brain` selects the provider: `file`, `mem`, `cloudflare`, `dynamo`, or
  `firestore`.
- `BrainCache.Directory` selects the local persistent cache path.
- Provider credentials stay in `conf/brains/<Brain>.yaml` under `BrainConfig`.
- Provider-sensitive sync settings live with the provider, not in `BrainCache`.
- The v3 engine always replays pending local outbox writes before command
  readiness; there is no configurable "start dirty" mode.
- Remove any custom `BrainCache.RequireRemoteCleanOnStartup` setting; it is no
  longer a valid v3 configuration key.

DynamoDB uses the AWS SDK default credential chain when `AccessKeyID` and
`SecretAccessKey` are omitted from `BrainConfig`. This includes local shared
credentials/profiles and EC2 instance credentials. If you configure static
DynamoDB credentials in custom config, store the encrypted values under custom
`conf/variables/*.yaml` `Secrets` and reference them from
`conf/brains/dynamo.yaml`:

```yaml
BrainConfig:
  TableName: "atlas-brain"
  Region: "us-east-2"
  AccessKeyID: {{ secret "DYNAMO_ACCESS_KEY_ID" | printf "%q" }}
  SecretAccessKey: {{ secret "DYNAMO_SECRET_ACCESS_KEY" | printf "%q" }}
```

Do not paste encrypted ciphertext directly into `AccessKeyID` or
`SecretAccessKey`; provider config receives the expanded plaintext value. If a
local profile or instance role should be used, leave those static keys out of
`BrainConfig` and make sure higher-priority `AWS_*` environment variables are
not accidentally overriding the desired source.

Cloudflare KV accepts these optional `BrainConfig` sync settings:

```yaml
BrainConfig:
  CloudWriteBudgetPerDay: 900
  CloudWriteMinIntervalMillis: 1100
  CoalesceWindowMillis: 2000
  FlushOnShutdownMaxMillis: 10000
  CheckpointVerifyRetries: 5
  CheckpointVerifyDelayMillis: 2000
```

The defaults are intended to keep normal Cloudflare KV use inside the free tier.
DynamoDB and Firestore currently use the engine cache defaults unless provider
config later grows provider-specific tuning.

The old file brain setting remains useful only for importing legacy local
memories:

```yaml
BrainConfig:
  BrainDirectory: state/brain
  Encode: true
```

After import, normal `Brain: file` runtime uses `BrainCache.Directory`.

### Startup, shutdown, and lock behavior

Cloud-backed v3 robots use `bot:instance-lock` as persistent ownership metadata.
Clean shutdown no longer deletes this memory. Instead, shutdown waits for
running pipelines, flushes pending normal brain writes, writes the lock as
`released` with the local cache database version, and then performs a final
brain flush before exit.

Startup accepts a released lock when the local cache is at least as new as the
released database version. If the released lock is newer than the local cache,
another cache has advanced the brain; run `gopherbot pull-brain` or point the
robot at the correct `BrainCache.Directory`.

If startup finds a held lock, it fails unless the lock matches the same local
cache nonce and active lock ID. That matching case is treated as same-VM crash
recovery: the new process reclaims its own previous lock, verifies the last
successful cloud write, replays the durable local outbox, and only then becomes
ready.

While startup is not ready to run commands, the robot responds to commands with:

```text
(the robot is still starting up, please wait and try your command again later)
```

### Cloud-backed upgrade paths

For an existing v2 robot using Cloudflare KV, DynamoDB, or Firestore:

1. Configure v3 with the same cloud provider credentials.
2. Run:

   ```bash
   gopherbot pull-brain
   ```

   This reads v2 or v3 cloud records and writes the local v3 cache. By default
   it does not modify the cloud provider.

3. Write v3 cloud records before starting the v3 runtime:

   ```bash
   gopherbot restore-brain
   ```

   Or combine import and cloud upgrade:

   ```bash
   gopherbot pull-brain -upgrade-cloud-v3
   ```

4. Start the v3 robot.

If a provider write budget stops the cloud upgrade partway through, the local
cache remains usable and complete. Re-run `restore-brain` with an appropriate
`-budget` or provider budget until all cloud records are v3.

### Local file brain upgrade

For a development robot that used the old file brain directory:

```bash
gopherbot pull-brain
```

This imports `BrainConfig.BrainDirectory` into `BrainCache.Directory`. Starting
with `Brain: file` then uses the local cache directly.

To later move that development robot to a cloud provider:

1. Change `Brain` and `conf/brains/<provider>.yaml` to the target cloud
   provider.
2. Run:

   ```bash
   gopherbot restore-brain
   ```

3. Start the v3 robot.

The local cache does not store the cloud driver identity. After the new remote
is populated with v3 records, startup safety is based on the remote lock
database version and checkpoint verification, not on whether the driver is
Cloudflare, DynamoDB, or Firestore.

### Rollback flexibility

The CLI can also write v2-compatible cloud data so a robot owner can deliberately
return to v2 code:

```bash
gopherbot restore-brain -v2
```

After this command, the remote brain is v2-compatible and is not valid for v3
runtime startup until it is written back as v3.

Useful command options:

- `pull-brain -dry-run` reports remote v2/v3 counts without writing.
- `pull-brain -force` replaces an existing local cache.
- `pull-brain -upgrade-cloud-v3` imports locally and writes upgraded v3 cloud
  records.
- `pull-brain -budget <n>` limits cloud writes for that run.
- `restore-brain` writes v3 cloud output by default.
- `restore-brain -v2` writes v2-compatible cloud output for deliberate rollback.
- `restore-brain -force` removes remote keys that are not present in the local
  cache.
- `restore-brain -dry-run` reports planned writes/removals.
- `restore-brain -budget <n>` limits cloud writes for that run.
- `fetch <key>` and `list` read the local cache by default.
- `fetch -validate-cloud <key>` checks the local cached record against the v3
  cloud record before printing the local value.
- `fetch -cloud <key>` reads directly from the v3 cloud brain; add
  `-update-cache` to repair an existing complete local cache for that key.
- `list -cloud` lists cloud keys.
- `store <key>` and `delete <key>` flush their cloud operation before reporting
  success.
- `flush-brain` drains queued cache writes to the configured cloud brain.

Any CLI command that touches the cloud prints local cache sync status to stderr.
Stdout remains reserved for command data such as fetched memory bytes or key
lists.

Normal robot startup contains no v2 conversion logic. If startup reports that a
cloud brain is v2/unversioned, run the CLI migration path rather than expecting
the daemon to repair cloud state automatically.

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

## 2026-06-24 Private Environment Precedence

Startup now treats `.env` and `private/environment` as private defaults. Values
already present in the launching process environment take precedence over values
from the private env file.

For example:

```bash
GOPHER_ENVIRONMENT=development gopherbot validate
```

uses `development` even if `.env` contains `GOPHER_ENVIRONMENT=production`.
Operators who relied on `.env` forcibly replacing exported shell variables
should unset those shell variables before startup.

## 2026-05-21 Thread Subscription Lifecycle Commands

Thread subscription callbacks now use engine-reserved command names:

- subscribed thread messages invoke the subscriber plugin as `_subscribed`
- subscription expiry invokes the subscriber plugin as `_expiresub`

Plugin configuration may no longer define command names beginning with `_`.
That prefix is reserved for engine lifecycle callbacks and `gopherbot validate`
will reject custom plugin configs that use it.

The optional `go-ai-prune` job has been removed. If a custom robot scheduled
`go-ai-prune`, remove that `ScheduledJobs` entry and any
`conf/jobs/go-ai-prune.yaml` override. The `ai-fallback` plugin now deletes
thread conversation memory when the engine expires the corresponding thread
subscription.

## 2026-05-20 Built-In Interpreter Context and Parameter Access

New-style built-in interpreter extensions (Lua, JS, Gopherbot Shell, Yaegi Go)
use `GetParameter` as the single API for all per-call context values and
configured pipeline parameters. `os.getenv("GOPHER_*")` does not work in Lua
or JS — this is a deliberate design choice, not a limitation.

### Legacy external scripts (bash, Python, Ruby)

External scripts receive all `GOPHER_*` context values as OS environment
variables. Configured pipeline parameters (from `ParameterSets`, namespace, or
`SetParameter`) are also present as environment variables unless
`SecureParameters: true` is set in `conf/robot.yaml`.

With `SecureParameters: true`:
- `GOPHER_*` context variables remain in the environment
- Configured parameters are withheld from the environment
- Scripts must call the Gopherbot HTTP JSON API (GetParameter) to retrieve individual
  parameters

### `HOME` and `PATH` preserve the launch environment

File-backed extensions no longer receive `HOME=$GOPHER_HOME` by default.
Instead, `HOME` and `PATH` are inherited from the parent process environment
when set, matching normal Unix subprocess behavior.

Use `GOPHER_HOME` for the robot home/root directory. This is the required
migration for scripts that previously used `$HOME` to find robot-owned files.
This change makes local development and host tools such as `kubectl`, `git`,
and cloud CLIs use the same home directory and command path as the process that
started the robot.

### Built-in interpreter extensions

| Runtime | Context values | Configured parameters |
|---|---|---|
| Lua | `bot:GetParameter("GOPHER_USER")` etc. | `bot:GetParameter("PARAM_NAME")` |
| JS | `bot.GetParameter("GOPHER_USER")` etc. | `bot.GetParameter("PARAM_NAME")` |
| GSH | `$GOPHER_USER` (env) or `GetParameter GOPHER_USER` | `GetParameter PARAM_NAME`¹ |
| Yaegi Go | `r.GetParameter("GOPHER_USER")` etc. | `r.GetParameter("PARAM_NAME")` |

¹ GSH follows `SecureParameters` for configured parameters: they are present
as shell env vars by default, but `GetParameter PARAM_NAME` must be used when
`SecureParameters: true`.

For Lua and JS, convenience fields on the bot object (`GBOT.user`,
`GBOT.channel`, `bot.user`, `bot.channel`, etc.) are pre-loaded from the call
context and remain idiomatic for the most common values. Less-common context
values — including `GOPHER_PRIVATE_COMMAND`, `GOPHER_CMDMODE`,
`GOPHER_INSTALLDIR`, `GOPHER_CONFIGDIR`, and `GOPHER_HOME` — are available via
`GetParameter`. `GOPHER_PRIVATE_COMMAND` returns `"true"` when the command was
addressed privately (DM or hidden-command context), empty string otherwise.

The `SecureParameters` setting has no effect on Lua, JS, or Yaegi Go: these
runtimes never receive parameters as environment variables regardless of that
setting.

Migration note for script rewrites:
- `os.getenv("GOPHER_USER")` → `bot:GetParameter("GOPHER_USER")` in Lua,
  `r.GetParameter("GOPHER_USER")` in Yaegi Go
- `os.getenv("GOPHER_HIDDEN_COMMAND")` has no equivalent for new-style plugins;
  use `RequiredPrivateCommands` / `RequireAllCommandsPrivate` in plugin config
  to enforce privacy, or check `bot:GetParameter("GOPHER_PRIVATE_COMMAND")`
  at runtime

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

## 2026-04-28 Privsep Child Process And UID-Only Separation

Privilege separation for file-backed extensions now uses one-shot child processes. The parent engine selects a child role, and `pipeline-child-exec` / `pipeline-child-rpc` commit to that role before running external scripts or built-in interpreters.

The earlier root `robot.yaml` supplementary-group keys have been removed:

```yaml
PrivsepAllowAllSupplementaryGroups: false
PrivsepAllowedSupplementaryGroups: []
```

Remove those keys from custom robot configuration before upgrading. If present, they fail startup through the normal unrecognized-key validation.

Privilege separation is now UID-only. Existing setuid privsep deployments must verify their install no longer sets the setgid bit.

Upgrade actions:

1. Install the binary owned by the unprivileged account, normally `nobody`, with setuid enabled and setgid disabled.
2. Prefer granting privileged host access directly to the invoking robot user by UID, not through broad groups such as `%wheel` or `%admin`.
3. Ensure `.env` is owner-readable only; current startup forces mode `0400` because it contains `GOPHER_ENCRYPTION_KEY`.
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

## 2026-05-26 HearSelf Is Now Engine-Owned

`HearSelf` is now a top-level `robot.yaml` setting and defaults to `true`.

Connector behavior is standardized:

- Connectors always forward messages they recognize as coming from the robot
  itself.
- Those messages are marked `ConnectorMessage.SelfMessage=true`.
- The engine decides whether to process or ignore them.

Migration guidance:

- Remove `ProtocolConfig.HearSelf` from connector config files such as
  `conf/protocols/ssh.yaml`, `conf/protocols/slack.yaml`, or terminal protocol
  overrides.
- To disable self-message processing for the whole robot, set:

```yaml
HearSelf: false
```

Existing self-trigger patterns continue to work by default. In particular,
`SelfMessage=true` messages can still trigger jobs, while normal plugin command
and ambient message matching remains skipped for self messages.

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
- `(alias) commands`: one-line summaries for commands available in the current channel
- `(alias) help-all`: one-line summaries including global commands
- `(alias) help <plugin>/<command>`: full help for one command, including generated SimpleMatcher options and optional multiline `Details`

Built-in unmatched-command fallback now returns algorithmic closest matches using the same command metadata and ranking logic.

## Engine-Owned Plugin Commands

The engine now permanently reserves plugin command names beginning with `_`.
Plugin configuration is rejected if any configured `Commands` or
`MessageMatchers` entry uses that prefix. Custom plugins should reserve `_`
commands for engine callbacks and keep user-facing commands unprefixed.

The standardized engine callback commands are:

- `_configure`
- `_init`
- `_authorize`
- `_usergroups`
- `_elevate`
- `_catchall`
- `_subscribed`
- `_expiresub`

Existing plugins will need to be updated to respond to the new engine-owned
commands, e.g. "configure" -> "_configure", "init" -> "_init".

For jobs, the internal command name is still engine-only and is not passed to
job handlers; job handlers receive only the job arguments.

## Authorizer `_usergroups` Contract (Help Filtering)

For group-aware help visibility, authorizer plugins can implement an optional:

- `_usergroups <username> <result_parameter>`

Expected behavior:

- return `Success` and set `result_parameter` via `SetParameter(...)` to a JSON array of group names, e.g. `["Helpdesk","Ops"]`
- return `NotFound` when group membership is intentionally unknown/indeterminate (for example, slow external policy checks)
- errors are treated the same as indeterminate membership for help filtering

Help/fallback filtering behavior:

- if `_usergroups` returns usable groups, commands requiring auth are filtered by `AuthRequire`
- if `_usergroups` is not implemented, returns `NotFound`, or errors, help output is not group-filtered (no-filter fallback)

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
