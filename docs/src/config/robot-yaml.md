# robot.yaml Reference

`robot.yaml` is the top-level configuration file for a robot. It chooses the robot's global runtime behavior: which connectors start, which brain and history providers are used, who the robot knows about, who can administer it, and which plugins, jobs, tasks, namespaces, and parameter sets are available.

Most detailed configuration does **not** belong in `robot.yaml`. In v3, `robot.yaml` normally contains selectors and robot-wide policy, while detailed provider and extension settings live in dedicated files:

- connector settings: `conf/protocols/<protocol>.yaml`
- brain settings: `conf/brains/<provider>.yaml`
- history settings: `conf/history/<provider>.yaml`
- queue settings: `conf/queues/<provider>.yaml`
- plugin settings: `conf/plugins/<plugin>.yaml`
- job settings: `conf/jobs/<job>.yaml`

For a custom robot, the file is usually `custom/conf/robot.yaml` inside the robot repository. Gopherbot first loads the installed defaults from the engine's `conf/robot.yaml`, then merges the custom robot's `conf/robot.yaml` over those defaults.

## Quick Example

This is the shape of a small custom `robot.yaml`:

```yaml
IgnoreUnlistedUsers: true
SecureParameters: true

{{ $environment := env "GOPHER_ENVIRONMENT" | default "production" }}
{{ printf "environments/%s.yaml" $environment | .Include }}

BotInfo:
  UserName: floyd
  Email: floyd@example.com
  FullName: Floyd Gopherbot
  FirstName: Floyd
  LastName: Gopherbot

Alias: ";"

DefaultMessageFormat: BasicMarkdown
DefaultJobChannel: general
HistoryProvider: file

TimeOuts:
  Plugin:
    Warn: 7m
    Kill: 14m
  Job:
    Warn: 1h
    Kill: 2h

AdminUsers:
- alice

UserRoster:
- UserName: alice
  Email: alice@example.com
  FullName: Alice Admin

ScheduledJobs:
- Name: pause-notifies
  Schedule: "0 0 8 * * *"
```

The included environment file commonly sets `PrimaryProtocol`, `DefaultProtocol`, `Brain`, and logging for a specific environment.

## Loading and Merge Rules

Like other Gopherbot config files, `robot.yaml` is expanded as a template before it is parsed as YAML. See [Config Templates](templates.md) for the common helper reference, include behavior, and secret/variable guidance.

See [Configuration Overview](file.md) for the full loading, merge, override, and `Append*` behavior.

Unknown top-level keys are startup errors. `BrainConfig`, `HistoryConfig`, and `QueueConfig` are also rejected in `robot.yaml`; put those in their dedicated provider files.

## Protocols and Routing

### PrimaryProtocol

Required.

`PrimaryProtocol` is the main connector protocol for the robot.

```yaml
PrimaryProtocol: ssh
```

The engine normalizes protocol names by trimming whitespace and lowercasing them. The primary protocol must have a matching connector config file at `conf/protocols/<protocol>.yaml`, and that file must contain a `ProtocolConfig` section.

Common values include:

- `ssh`
- `slack`
- `googlechat`
- `terminal`
- `test`
- `nullconn`

The `terminal` connector is useful for local work, but it is not supported as a secondary protocol.

During reload, changing `PrimaryProtocol` is ignored once a connector runtime is already active. Restart the robot to change the primary protocol.

### DefaultProtocol

Optional. Defaults to `PrimaryProtocol`.

`DefaultProtocol` is used when the robot sends an outbound message without an incoming message context, such as plugin initialization or scheduled work.

```yaml
DefaultProtocol: ssh
```

If set, it must be either the primary protocol or one of `SecondaryProtocols`. If it is not, Gopherbot logs a warning and falls back to the primary protocol.

### SecondaryProtocols

Optional.

`SecondaryProtocols` lists additional connector protocols to initialize and run alongside the primary protocol.

```yaml
SecondaryProtocols:
- slack
- googlechat
```

Each secondary protocol should have `conf/protocols/<protocol>.yaml`. Secondary connector startup failures are logged, but they do not stop the primary connector from running.

Rules:

- duplicates are ignored
- the primary protocol is ignored if repeated here
- `terminal` is ignored as a secondary protocol
- protocol names are normalized internally

### ChannelRoster

Optional.

`ChannelRoster` maps channel names to protocol channel IDs when a connector cannot provide a stable human-readable name, or when you want configuration to refer to a friendly channel name.

```yaml
ChannelRoster:
- ChannelName: general
  ChannelID: C123456
```

Fields:

- `ChannelName`: name used in robot config
- `ChannelID`: connector's internal channel ID

Top-level `ChannelRoster` entries apply to the primary protocol. Protocol files can also provide `ChannelRoster`; those entries are tagged with that protocol as they load. Primary-protocol entries from `robot.yaml` are appended after the primary protocol file so robot-specific channel mappings can override protocol-specific lookup maps.

### JoinChannels

Optional.

`JoinChannels` lists channels the robot should try to join during startup.

```yaml
JoinChannels:
- general
- operations
```

Gopherbot also tries to join channels from `DefaultChannels` and `DefaultJobChannel`. Not every connector supports joining channels programmatically.

### DefaultChannels

Optional.

`DefaultChannels` defines the channels where plugins are available when the plugin's own config does not specify `Channels` and does not set `AllChannels`.

```yaml
DefaultChannels:
- general
- random
```

Plugin-specific channel configuration belongs in `conf/plugins/<plugin>.yaml`.

## Robot Identity and Addressing

### BotInfo

Recommended.

`BotInfo` defines the robot's canonical identity. Connectors receive this information during initialization, and plugin APIs use it for bot attributes.

```yaml
BotInfo:
  UserName: floyd
  Email: floyd@example.com
  FullName: Floyd Gopherbot
  FirstName: Floyd
  LastName: Gopherbot
```

Fields:

- `UserName`: the name users type when addressing the robot, such as `floyd, ping`
- `Email`: used by email-sending APIs as the sender address
- `FullName`: human-readable robot name
- `FirstName`: optional first name attribute
- `LastName`: optional last name attribute

`GetBotAttribute("name")` returns `BotInfo.UserName`. `GetBotAttribute("email")` returns `BotInfo.Email`. `GetBotAttribute("fullName")` returns `BotInfo.FullName`.

### Alias

Optional.

`Alias` is a one-character shortcut for addressing the robot.

```yaml
Alias: ";"
```

With this example, users can say `;ping` instead of `floyd, ping`.

Allowed alias characters are:

```text
* + ^ $ ? \ [ ] { } & ! ; : - % # @ ~ < >
```

Only the first rune of the string is used. If the alias is invalid, startup fails.

### HearSelf

Optional. Defaults to `true`.

`HearSelf` controls whether the engine processes messages that a connector marks as originating from the robot itself.

```yaml
HearSelf: false
```

Connectors still own the decision to mark a message as a self-message. This option only controls whether the engine ignores those marked messages.

## Users and Access Policy

### UserRoster

Optional, but strongly recommended.

`UserRoster` is the global user directory. Gopherbot's authorization and policy decisions are based on canonical usernames from this directory, not on connector-specific transport IDs.

```yaml
UserRoster:
- UserName: alice
  Email: alice@example.com
  FullName: Alice Admin
  FirstName: Alice
  LastName: Admin
  Phone: "+1-555-0100"
- UserName: buildbot
  BotUser: true
```

Fields:

- `UserName`: canonical username used in config, authorization, memory ownership, and policy checks
- `Email`: user email returned by attribute APIs
- `Phone`: user phone returned by attribute APIs
- `FullName`: full display name returned by attribute APIs
- `FirstName`: first name returned by attribute APIs
- `LastName`: last name returned by attribute APIs
- `BotUser`: marks an automated user; bot users are not checked against ambient message matchers and do not fall through to catch-all plugins
- `UserID`: legacy parse-only field; accepted so older configs can load, but ignored by the engine

`UserName` must be lowercase. Entries with an empty username or uppercase letters are ignored.

Connector-specific user IDs do **not** belong in top-level `UserRoster`. Put protocol-local identity mapping in the connector's config file, such as `conf/protocols/slack.yaml`, `conf/protocols/googlechat.yaml`, or `conf/protocols/ssh.yaml`.

### IgnoreUnlistedUsers

Optional. Defaults to `false` unless set by custom config.

When `IgnoreUnlistedUsers` is true, Gopherbot drops incoming messages unless the connector has validated the transport user and resolved it to a canonical username listed in `UserRoster`.

```yaml
IgnoreUnlistedUsers: true
```

This is the large security switch for roster-based deployments. It is enforced before any worker or pipeline is created.

### IgnoreUsers

Optional.

`IgnoreUsers` lists canonical usernames the robot should never respond to.

```yaml
IgnoreUsers:
- otherbot
- slackbot
```

The comparison is case-insensitive and happens before any pipeline is created.

### AdminUsers

Optional. Defaults to an empty list if not configured.

`AdminUsers` lists canonical usernames with access to built-in administrative commands.

```yaml
AdminUsers:
- alice
- david
```

Admin checks are username-based. Connector-provided flags, message content, and user-modifiable runtime state do not make a user an admin.

Scheduled jobs run as automatic tasks and are treated as admin by design because they are scheduled by configuration.

### DefaultAuthorizer

Optional.

`DefaultAuthorizer` names the authorizer plugin used when a plugin or job requires authorization but does not specify its own `Authorizer`.

```yaml
DefaultAuthorizer: groups
```

Plugin and job authorization rules live in `conf/plugins/<plugin>.yaml` and `conf/jobs/<job>.yaml`.

### DefaultElevator

Optional.

`DefaultElevator` names the elevation plugin used when a plugin command requires elevation but does not specify its own `Elevator`.

```yaml
DefaultElevator: builtin-userapproval
```

Elevation is checked after base authorization succeeds. For the built-in human approval elevator, see [User Approval Elevation](../security/userapproval.md).

## Brain, History, and Queues

### Brain

Optional. Defaults to `mem` if unset at brain initialization, though installed defaults normally set a value.

`Brain` selects the provider used for long-term robot memory.

```yaml
Brain: file
```

Common values:

- `mem`: in-memory brain
- `file`: local cached file brain
- `dynamo`: DynamoDB-backed remote brain
- `cloudflare`: Cloudflare KV-backed remote brain
- `firestore`: Firestore-backed remote brain

Provider-specific settings belong in `conf/brains/<provider>.yaml` under `BrainConfig`.

For `file`, Gopherbot uses the v3 local brain cache. If legacy file-brain data is present in the old brain directory and the v3 cache has not been initialized, startup asks you to run `gopherbot pull-brain`.

### BrainCache

Optional.

`BrainCache` configures the engine-owned local cache used by the v3 brain system.

```yaml
BrainCache:
  Directory: state/brain-cache
```

Fields:

- `Directory`: directory for local brain cache state

If omitted, the directory defaults to `state/brain-cache`. Installed defaults normally derive this from `GOPHER_STATE_DIRECTORY`.

### HistoryProvider

Optional. Defaults to `mem`.

`HistoryProvider` selects where plugin and job run logs are stored.

```yaml
HistoryProvider: file
```

Common values:

- `mem`: in-memory history
- `file`: file-backed history

Provider-specific settings belong in `conf/history/<provider>.yaml` under `HistoryConfig`.

If the configured provider is not registered or returns nil during startup, Gopherbot logs an error and falls back to `mem`.

### QueueProviders

Optional.

`QueueProviders` lists queue providers that should start after full robot initialization.

```yaml
QueueProviders:
- gcloud
```

Provider names are trimmed, lowercased, and deduplicated. Provider-specific settings belong in `conf/queues/<provider>.yaml` under `QueueConfig`.

Queue provider startup failures are logged per provider and do not stop the robot from running. Jobs opt in to queue triggering with job-level `UUIDTrigger` in `conf/jobs/<job>.yaml`.

## Messages and Formatting

### DefaultMessageFormat

Optional. Defaults to `BasicMarkdown`.

`DefaultMessageFormat` controls how outgoing messages are formatted when a plugin, job, or built-in send path does not choose a format explicitly.

```yaml
DefaultMessageFormat: BasicMarkdown
```

Supported values:

- `BasicMarkdown`
- `Raw`
- `Fixed`
- `Variable`

`BasicMarkdown` is the v3 default portable format. `Raw` preserves protocol-native behavior for older robots that need it. Unknown values log an error and fall back to `BasicMarkdown`.

### DefaultJobChannel

Optional, but recommended.

`DefaultJobChannel` is the default channel for job status messages and other operator-facing output when a job or alert does not specify a channel.

```yaml
DefaultJobChannel: general
```

Jobs without their own `Channel` and without a `DefaultJobChannel` can be disabled during config loading because Gopherbot has nowhere to report their activity.

### ReadyMessage

Optional.

`ReadyMessage` sends a channel message after startup readiness opens.

```yaml
ReadyMessage: "floyd is ready"
```

If unset or blank, no ready message is sent.

### ReadyChannel

Optional. Defaults to `DefaultJobChannel`.

`ReadyChannel` chooses where `ReadyMessage` is sent.

```yaml
ReadyChannel: general
```

If `ReadyMessage` is configured but neither `ReadyChannel` nor `DefaultJobChannel` is available, Gopherbot logs an error and skips the message.

## Scheduling and Time

### ScheduledJobs

Optional.

`ScheduledJobs` schedules configured jobs.

```yaml
ScheduledJobs:
- Name: pause-notifies
  Schedule: "0 0 8 * * *"
- Name: install-libs
  Schedule: "@init"
- Name: hello
  Schedule: "@every 30s"
  Arguments:
  - "Hello"
  - "World"
```

Fields:

- `Name`: name of the job to run
- `Schedule`: cron expression or descriptor
- `Arguments`: optional arguments passed to the job
- `Command`: accepted by the shared task spec, mainly useful for plugin-like task specs

Schedules use `github.com/robfig/cron/v3`. Gopherbot accepts seconds as an optional first field, so both five-field and six-field cron expressions can be used. Descriptors such as `@every 30s` and `@init` are supported.

`@init` jobs run during startup before normal plugin initialization. Scheduled jobs run as automatic tasks and bypass normal user authorization/elevation checks because they are controlled by robot configuration.

Only jobs can be scheduled. Disabled jobs are skipped.

### TimeZone

Optional.

`TimeZone` sets the time zone used for scheduled jobs.

```yaml
TimeZone: America/New_York
```

Use an IANA time zone name. If the value cannot be loaded, Gopherbot logs an error and uses the system default time zone.

### TimeOuts

Optional.

`TimeOuts` sets default warning and kill thresholds for plugin and job pipelines.

```yaml
TimeOuts:
  Plugin:
    Warn: 7m
    Kill: 14m
  Job:
    Warn: 1h
    Kill: 2h
```

Fields:

- `Plugin.Warn`: how long a plugin can run before an operator warning
- `Plugin.Kill`: how long a plugin can run before termination
- `Job.Warn`: how long a job can run before an operator warning
- `Job.Kill`: how long a job can run before termination

Durations should normally be quoted or unquoted Go duration strings such as `30s`, `7m`, or `1h`. Integer values are accepted as nanoseconds.

Rules:

- negative durations are invalid
- if both `Warn` and `Kill` are set, `Kill` must be greater than `Warn`
- zero disables that threshold

Plugin, job, and task configs can override these defaults with their own `TimeOuts` block.

## Plugins, Jobs, Tasks, Namespaces, and Parameters

`robot.yaml` declares which extensions exist. Detailed plugin and job behavior should usually live in `conf/plugins/<plugin>.yaml` and `conf/jobs/<job>.yaml`.

The root maps in this section all use the map key as the extension or set name:

```yaml
ExternalPlugins:
  "my-plugin":
    Path: plugins/my-plugin.gsh
    Description: My custom command
```

### Common Declaration Fields

These fields are used in `ExternalPlugins`, `ExternalJobs`, `ExternalTasks`, `GoPlugins`, `GoJobs`, `GoTasks`, `NameSpaces`, and `ParameterSets`. Not every field is meaningful for every map.

- `Path`: executable or source path for external extensions
- `Description`: human-readable description
- `NameSpace`: namespace for shared memory and parameters
- `ParameterSets`: list of named parameter sets to attach
- `Disabled`: disables the entry
- `Homed`: runs with the robot home as the working context
- `Privileged`: privilege setting for the extension
- `Parameters`: fixed parameters made available to the extension

Parameter item shape:

```yaml
Parameters:
- Name: WEATHER_API_KEY
  Value: '{{ secret "WEATHER_API_KEY" }}'
```

Security note: with `SecureParameters: true`, extensions must retrieve parameters through the Robot API instead of receiving them as environment variables.

### ExternalPlugins

Optional.

`ExternalPlugins` declares executable or interpreted plugins that can respond to user commands.

```yaml
ExternalPlugins:
  "admin":
    Path: plugins/admin.gsh
    Description: Administrative plugin
    Privileged: true
```

External plugins default to `Privileged: false` when `Privileged` is omitted.

Plugin command matchers, channel restrictions, help metadata, authorization, elevation, and plugin-specific config usually belong in `conf/plugins/<plugin>.yaml`.

### GoPlugins

Optional.

`GoPlugins` declares compiled Go plugins registered with the engine.

```yaml
GoPlugins:
  "help":
    Description: Help plugin
```

Go plugins default to `Privileged: false` when `Privileged` is omitted.

### ExternalJobs

Optional.

`ExternalJobs` declares executable or interpreted jobs.

```yaml
ExternalJobs:
  "nightly-report":
    Path: jobs/nightly-report.gsh
    Description: Build the nightly report
```

External jobs default to `Privileged: true` when `Privileged` is omitted.

Job triggers, channel, arguments, quiet/log settings, and job-specific config usually belong in `conf/jobs/<job>.yaml`.

### GoJobs

Optional.

`GoJobs` declares compiled Go jobs registered with the engine.

```yaml
GoJobs:
  "go-bootstrap":
    Description: Bootstrap a custom robot repository
```

Go jobs default to `Privileged: true` when `Privileged` is omitted.

### ExternalTasks

Optional.

`ExternalTasks` declares reusable pipeline tasks. Tasks are not command starters by themselves; plugins and jobs can add them to a pipeline.

```yaml
ExternalTasks:
  "render-report":
    Path: tasks/render-report.gsh
    Description: Render a report artifact
```

External tasks default to `Privileged: false` when `Privileged` is omitted. A privileged task cannot be added to an unprivileged pipeline; `Privileged: true` on a task means the task requires a privileged pipeline, not that the task can elevate the pipeline. For example, a task that runs `kubectl` with a robot-managed kubeconfig should be marked privileged and added only by a privileged job or plugin.

### GoTasks

Optional.

`GoTasks` declares compiled Go tasks registered with the engine.

```yaml
GoTasks:
  "ssh-agent":
    Description: Prepare SSH agent state
```

Go task entries pass through even when disabled because compiled tasks are enabled by default and may need an explicit `Disabled: true`.

### NameSpaces

Optional.

`NameSpaces` declares shared namespaces for memory and parameters.

```yaml
NameSpaces:
  "deploy":
    Description: Shared deployment parameters
    Parameters:
    - Name: DEPLOY_ENV
      Value: production
```

An extension can use `NameSpace: deploy` to share long-term memories and parameters with other extensions using the same namespace.

### ParameterSets

Optional.

`ParameterSets` declares named sets of parameters that can be attached to plugins, jobs, tasks, or identity providers.

```yaml
ParameterSets:
  "github_oauth":
    Description: GitHub OAuth client credentials
    Parameters:
    - Name: CLIENT_ID
      Value: '{{ variable "GITHUB_CLIENT_ID" }}'
    - Name: CLIENT_SECRET
      Value: '{{ secret "GITHUB_CLIENT_SECRET" }}'
```

Attach a set to an extension with `ParameterSets`:

```yaml
ExternalPlugins:
  "github-link":
    ParameterSets:
    - github_oauth
```

## Identity Providers

### IdentityProviders

Optional.

`IdentityProviders` configures providers used by the `GetIdentityCredential` Robot API for user-linked credentials and refresh behavior.

```yaml
IdentityProviders:
  github:
    Type: oauth2
    CredentialParameterSet: github_oauth
    OAuth2:
      TokenEndpointAuthMethod: client_secret_post
      Token:
        URL: https://github.com/login/oauth/access_token
        Headers:
          Accept: application/json
```

Provider keys are trimmed and lowercased.

Fields:

- `Type`: provider type. Current user-linked refresh support is OAuth2-oriented.
- `CredentialParameterSet`: name of a `ParameterSets` entry containing provider credentials such as `CLIENT_ID` and `CLIENT_SECRET`
- `OAuth2`: OAuth2 refresh configuration

`OAuth2` fields:

- `TokenEndpointAuthMethod`: token endpoint auth method, such as `client_secret_post`
- `Token.URL`: token endpoint URL
- `Token.Headers`: headers sent to the token endpoint
- `Token.Parameters`: parameters sent to the token endpoint
- `Token.RefreshParameters`: additional parameters used during refresh

If an identity provider references a missing parameter set, Gopherbot logs an error. The provider is still loaded into the registry, but API calls can fail later if required settings are missing.

Secret boundary: identity-provider secrets should be supplied through the referenced `ParameterSets`. Unprivileged extensions do not get broad access to provider registries or shared secret-bearing configuration.

## Logging, HTTP, and Runtime Directories

### LocalPort

Optional. Defaults to `0`.

`LocalPort` controls the localhost HTTP/JSON API listener used by external plugins.

```yaml
LocalPort: 0
```

`0` means choose the first available port. The listener binds to `127.0.0.1:<LocalPort>`. CLI-only operations do not start this listener.

### HttpDebug

Optional. Defaults to `false`.

`HttpDebug` enables debug logging for local HTTP JSON API requests and replies.

```yaml
HttpDebug: true
```

Use this only for low-level debugging. It does not scrub secrets. Installed defaults also force debug-friendly logging when `GOPHER_HTTP_DEBUG=true`.

### LogDest

Optional.

`LogDest` chooses where the robot log is written.

```yaml
LogDest: robot.log
```

Common values:

- `stdout`
- `stderr`
- a filename such as `robot.log`

Installed defaults avoid logging terminal-connector UI output to stdout.

### LogLevel

Optional.

`LogLevel` sets the initial log level.

```yaml
LogLevel: info
```

Recognized values:

- `trace`
- `debug`
- `info`
- `audit`
- `warn`
- `error`

Unknown values resolve to `error`. Runtime admin commands can change the log level after startup.

### WorkSpace

Optional.

`WorkSpace` is the read/write directory the robot uses for work.

```yaml
WorkSpace: workspace
```

If the directory can be created or opened, Gopherbot uses it as the workspace. If it cannot, Gopherbot logs an error and uses the robot config directory instead.

## Email

### MailConfig

Optional.

`MailConfig` configures SMTP delivery for the Robot email APIs.

```yaml
MailConfig:
  Mailhost: smtp.example.com:25
  Authtype: plain
  User: floyd
  Password: '{{ secret "SMTP_PASSWORD" }}'
```

Fields:

- `Mailhost`: SMTP host, normally with port
- `Authtype`: `plain` for SMTP plain auth; any other value sends without auth
- `User`: SMTP username for `plain`
- `Password`: SMTP password for `plain`

The robot's sender address comes from `BotInfo.Email`. User recipient addresses come from `UserRoster`, protocol-provided attributes, or explicit email addresses supplied by the caller.

### AdminContact

Optional.

`AdminContact` is informational contact text exposed through bot attributes and status/help paths.

```yaml
AdminContact: "Ops Team <ops@example.com>"
```

### Legacy Email and Name Keys

`Email` and `Name` are accepted as top-level keys by the v3 loader for compatibility, but they are not applied to runtime robot identity. Use `BotInfo.Email` and `BotInfo.UserName` instead.

## Security and Secrets

### SecureParameters

Optional. Defaults to `false` unless set by custom config.

When `SecureParameters` is true, Gopherbot does not publish configured parameters as environment variables to jobs, plugins, and tasks. Extensions must use the Robot API to retrieve parameters explicitly.

```yaml
SecureParameters: true
```

This is recommended for robots that handle secrets.

### EncryptionKey

Legacy/compatibility option.

`EncryptionKey` is an older config path for unlocking the real encryption key from the brain.

```yaml
EncryptionKey: "a-string-at-least-32-bytes-long"
```

Modern v3 robots should prefer `GOPHER_ENCRYPTION_KEY` plus the binary encrypted key file generated by `gopherbot genkey` or startup initialization. During config dumps, Gopherbot masks this value as `XXXXXX`.

### Variables and Secrets Files

Do not put raw shared secrets directly in `robot.yaml`. Put encrypted secrets and plaintext variables in:

- `custom/conf/variables/common.yaml`
- `custom/conf/variables/<environment>.yaml`

Example:

```yaml
Secrets:
  SMTP_PASSWORD: "<encrypted-base64-value>"
Variables:
  GITHUB_CLIENT_ID: "client-id"
```

Then reference them from config templates:

```yaml
Password: '{{ secret "SMTP_PASSWORD" }}'
Value: '{{ variable "GITHUB_CLIENT_ID" }}'
```

Environment-specific values override common values.

## Provider and Connector Config That Must Not Be in robot.yaml

These top-level keys are invalid in `robot.yaml`:

- `BrainConfig`
- `HistoryConfig`
- `QueueConfig`

Put them in:

- `conf/brains/<provider>.yaml`
- `conf/history/<provider>.yaml`
- `conf/queues/<provider>.yaml`

Connector `ProtocolConfig` also belongs in `conf/protocols/<protocol>.yaml`, not in `robot.yaml`.

Top-level `UserMap` is specifically rejected from protocol files in v3. Connector-local identity mapping must live under that connector's `ProtocolConfig`, using the fields documented for that connector, such as Slack and Google Chat `ProtocolConfig.UserMap` or SSH `ProtocolConfig.UserKeys`.

## Complete Top-Level Key Index

This index lists the top-level keys accepted by the v3 `robot.yaml` loader.

| Key | Purpose |
| --- | --- |
| `AdminContact` | Informational robot administrator contact |
| `AdminUsers` | Canonical usernames with admin access |
| `Alias` | One-character command-addressing shortcut |
| `BotInfo` | Robot identity |
| `Brain` | Brain provider selector |
| `BrainCache` | Local v3 brain cache settings |
| `ChannelRoster` | Channel name/ID mappings |
| `DefaultAuthorizer` | Default authorization plugin |
| `DefaultChannels` | Default plugin channels |
| `DefaultElevator` | Default elevation plugin |
| `DefaultJobChannel` | Default channel for job/operator output |
| `DefaultMessageFormat` | Default outgoing message format |
| `DefaultProtocol` | Protocol for context-free outbound messages |
| `Email` | Legacy accepted key; use `BotInfo.Email` |
| `EncryptionKey` | Legacy encryption unlock setting |
| `ExternalJobs` | External job declarations |
| `ExternalPlugins` | External plugin declarations |
| `ExternalTasks` | External pipeline task declarations |
| `GoJobs` | Compiled Go job declarations |
| `GoPlugins` | Compiled Go plugin declarations |
| `GoTasks` | Compiled Go task declarations |
| `HearSelf` | Process or ignore connector-marked self messages |
| `HistoryProvider` | History provider selector |
| `HttpDebug` | Debug local HTTP API traffic |
| `IdentityProviders` | User-linked identity provider registry |
| `IgnoreUnlistedUsers` | Drop unvalidated/unlisted users before pipelines |
| `IgnoreUsers` | Users the robot never responds to |
| `JoinChannels` | Channels to join during startup |
| `LocalPort` | Local HTTP/JSON API port |
| `LogDest` | Log destination |
| `LogLevel` | Initial log level |
| `MailConfig` | SMTP settings |
| `Name` | Legacy accepted key; use `BotInfo.UserName` |
| `NameSpaces` | Shared memory/parameter namespaces |
| `ParameterSets` | Reusable named parameter sets |
| `PrimaryProtocol` | Required primary connector protocol |
| `QueueProviders` | Queue provider selectors |
| `ReadyChannel` | Channel for startup ready message |
| `ReadyMessage` | Optional message after startup readiness |
| `ScheduledJobs` | Job schedules |
| `SecondaryProtocols` | Additional connector protocols |
| `SecureParameters` | Require API parameter retrieval instead of env vars |
| `TimeOuts` | Default plugin/job timeout thresholds |
| `TimeZone` | Time zone for scheduled jobs |
| `UserRoster` | Canonical user directory |
| `WorkSpace` | Robot working directory |
