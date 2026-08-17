# plugins/&lt;plugin&gt;.yaml Reference

`conf/plugins/<plugin>.yaml` configures how a plugin is matched, where it is visible, what policy applies to its commands, and what plugin-specific `Config` data the plugin can read at runtime.

It does **not** declare that the plugin exists. Plugin declarations live in `robot.yaml` under `ExternalPlugins` or `GoPlugins`.

```yaml
ExternalPlugins:
  "hello":
    Description: Example hello-world plugin
    Path: plugins/hello.lua
```

After a plugin is declared, put command matchers and plugin behavior in:

```text
custom/conf/plugins/hello.yaml
```

## Quick Example

```yaml
Channels:
- general
- operations

AllowedPrivateCommands:
- status

Commands:
- Command: status
  SimpleMatcher: "service status <service:ident>"
  Usage: "service status <service>"
  Summary: "show current status for a named service"
  Examples:
  - "(alias) service status api"
  Keywords: [ "service", "status", "health" ]

Config:
  DefaultRegion: us-east-1
```

The plugin handler receives `status` as the command and `api` as the first argument.

## Loading and Merge Rules

Plugin config is loaded from:

```text
conf/plugins/<plugin>.yaml
```

Gopherbot loads installed plugin defaults first, then merges custom robot overrides. Maps merge recursively, arrays replace by default, and arrays append only when the key starts with `Append`.

Plugin config files are templates like the rest of the config tree. See [Config Templates](templates.md).

For compiled Go plugins and interpreter-backed plugins that provide a default configuration through their configure hook, that default is also part of the plugin's base config before installed and custom YAML overrides are applied.

Unknown top-level keys disable the plugin at load time. Some declaration keys are valid in `robot.yaml` but intentionally invalid or ignored in `conf/plugins/<plugin>.yaml`; see [What Belongs in robot.yaml](#what-belongs-in-robotyaml).

## What Belongs in robot.yaml

Put plugin declaration and execution identity in `robot.yaml`, not in `conf/plugins/<plugin>.yaml`.

Use `robot.yaml` for:

- `Path`
- `Description`
- `NameSpace`
- `ParameterSets`
- `Parameters`
- `Homed`
- `Privileged`
- `Disabled` when you want to disable a plugin before plugin config loading

Use `conf/plugins/<plugin>.yaml` for:

- command matchers
- channel visibility
- user/admin/authorization/elevation policy
- private-command policy
- catch-all behavior
- reply matchers
- timeout overrides
- plugin-specific `Config`

Important implementation details:

- `Privileged` is illegal in `conf/plugins/<plugin>.yaml`; set it in `robot.yaml`.
- `NameSpace` and `ParameterSets` in plugin YAML are logged and ignored; set them in `robot.yaml`.
- `Path`, `Description`, `Parameters`, and `Homed` belong in `robot.yaml`. They are not useful plugin-YAML overrides.

## Command Matching

### Commands

`Commands` are directed commands. They are checked when the user addresses the robot by name, alias, DM, slash/private route, or another connector-owned bot-message path.

```yaml
Commands:
- Command: ping
  SimpleMatcher: "ping"
  Usage: "ping"
  Summary: "see if the bot is alive"
  Examples:
  - "(alias) ping"
  Keywords: [ "ping" ]
```

Each command matcher must have:

- `Command`: the command token passed to the plugin
- exactly one of `Regex` or `SimpleMatcher`

`Command` names must not start with `_`. Leading underscore commands are reserved for engine lifecycle callbacks such as `_init`, `_configure`, `_authorize`, `_usergroups`, `_elevate`, `_catchall`, `_subscribed`, and `_expiresub`.

### Regex

`Regex` is a Go regular expression for the command body after the robot addressing prefix has been removed.

```yaml
Commands:
- Command: deploy
  Regex: '(?i:deploy ([A-Za-z0-9._/-]+))'
```

For `Commands`, Gopherbot wraps the regex with leading/trailing whitespace tolerance and anchors it as an exact match. Do not add surrounding `^` and `$` unless you specifically intend to anchor inside the command body.

Capture groups become positional arguments to the plugin.

### SimpleMatcher

`SimpleMatcher` is the preferred syntax for common directed commands. This section gives the shape; see the [SimpleMatcher reference](../reference/SimpleMatcher.md) for the complete grammar, capture semantics, typed values, option blocks, diagnostics, and migration guidance.

```yaml
Commands:
- Command: loglevel
  SimpleMatcher: "set log level {to} (level:trace|debug|info|warn|error)"
```

Simple matchers are case-insensitive and tolerate extra whitespace. Spaces between command words can be typed as spaces or dashes, but one contiguous command phrase must use one separator style consistently. Spaces before typed captures and options require whitespace. For example, `spot (type:rails|devops) up` matches `spot-devops-up`, while `rails up [<branch:token>]` matches `rails-up dev`, but not `rails-up-dev`.

Common forms:

- `literal text`: required non-capturing words
- `{optional noise}`: optional non-capturing words
- `/start|begin/`: required non-capturing synonyms
- `(label:a|b|c)`: required capturing choice
- `[label:a|b|c]`: optional capturing choice
- `[-options:-spot|-branch:<token>]`: optional argv-style options block; matched options become individual args
- `<name:ident>`: typed capture
- `[<name:rest>]`: optional typed capture

Options blocks emit no argument when omitted. Captures after an options block are variable-position, so plugins using them should parse leading `-` arguments before interpreting the remaining positional arguments.

Built-in capture types include `ident`, `token`, `rest`, `number`, `decimal`, `bool`, `duration`, `email`, `url`, `ip`, `ipv4`, `ipv6`, `cidr`, `dnsname`, `slug`, and `base64`.

SimpleMatcher is supported only for directed `Commands`. It is rejected for `MessageMatchers`, `ReplyMatchers`, and job argument matchers.

### Command Matcher Fields

These fields are accepted inside each `Commands` item:

- `Command`: command token passed to the plugin
- `Regex`: regular expression matcher
- `SimpleMatcher`: simplified matcher syntax
- `Usage`: short usage text for help
- `Summary`: one-sentence help summary
- `Examples`: example invocations
- `Keywords`: help search terms
- `ChannelOnly`: match only normal channel messages, not threaded messages
- `Contexts`: capture-group labels for short-term "it"/context memory

`Usage`, `Summary`, `Examples`, and `Keywords` drive the v3 help system. They do not change matching behavior.

### Contexts

`Contexts` lets a command remember a captured value for later commands that use a generic reference such as `it`.

```yaml
Commands:
- Command: show-ticket
  SimpleMatcher: "show ticket <ticket:slug>"
  Contexts:
  - ticket:it:that
- Command: close-ticket
  SimpleMatcher: "close ticket <ticket:slug>"
  Contexts:
  - ticket:it:that
```

When a user says `show ticket OPS-123`, the `ticket` context can remember `OPS-123`. If the next command says `close ticket it`, the engine can substitute the remembered value. If no value is remembered, the user is asked to be more specific.

## Ambient Message Matching

### MessageMatchers

`MessageMatchers` listen to messages that were not necessarily addressed to the robot.

```yaml
MessageMatchers:
- Command: help
  Regex: '^(?i:help)$'
```

Fields are the same `InputMatcher` shape as `Commands`, but with two important differences:

- `SimpleMatcher` is not supported
- Gopherbot does not add anchors around `MessageMatchers`; include `^` and `$` yourself when you want exact matching

`MessageMatchers` can be powerful and noisy. Use them sparingly.

### AmbientMatchCommand

Optional. Defaults to `false`.

When `AmbientMatchCommand` is false, `MessageMatchers` are skipped for messages that were already recognized as directed robot commands.

```yaml
AmbientMatchCommand: true
```

Set this only when the plugin should inspect both ambient messages and directed commands.

### MatchUnlisted

Optional. Defaults to `false`.

When `MatchUnlisted` is false, ambient `MessageMatchers` are skipped for users that are not listed in the global `UserRoster`.

```yaml
MatchUnlisted: true
```

This affects ambient matching only. It does not bypass `IgnoreUnlistedUsers`, which is a pre-pipeline gate in `robot.yaml`.

## Channel Visibility

### Channels

`Channels` lists the public channels where the plugin is available.

```yaml
Channels:
- general
- operations
```

If `Channels` is empty, Gopherbot uses `DefaultChannels` from `robot.yaml` when that list is configured.

### AllChannels

`AllChannels: true` makes the plugin available in every normal channel where the robot is present.

```yaml
AllChannels: true
```

If neither plugin `Channels` nor robot `DefaultChannels` are configured, Gopherbot defaults the plugin to all channels unless `AllChannels` was explicitly set to false.

### Channel

`Channel` is accepted for plugins because plugins can also be scheduled or used in pipeline contexts that need a channel.

```yaml
Channel: general
```

For normal command visibility, use `Channels` or `AllChannels`.

### Direct Messages

Plugins are generally considered available in direct messages for matching purposes. Whether a command can actually run in a private context is controlled by the private-command policy below.

## Private Commands

Private contexts include direct messages and connector-supported hidden/ephemeral messages such as slash-command routes.

### AllowedPrivateCommands

`AllowedPrivateCommands` lists commands that may run in private contexts.

```yaml
AllowedPrivateCommands:
- status
- whoami
```

Use `*` to allow every command in the plugin to run privately:

```yaml
AllowedPrivateCommands:
- "*"
```

### RequiredPrivateCommands

`RequiredPrivateCommands` lists commands that must run in a private context.

```yaml
RequiredPrivateCommands:
- dump
- token
```

If a required-private command is invoked publicly, the engine rejects it before plugin code runs.

### RequireAllCommandsPrivate

`RequireAllCommandsPrivate: true` requires every command in the plugin to run privately.

```yaml
RequireAllCommandsPrivate: true
```

### RestrictPrivateChannels

`RestrictPrivateChannels: true` makes configured `Channels` an access boundary for private-capable commands.

```yaml
Channels:
- security
RestrictPrivateChannels: true
AllowedPrivateCommands:
- rotate-key
```

Without this setting, a private-capable command can run in DMs or hidden contexts even if the plugin has public channel restrictions. With this setting:

- DMs are rejected because a DM does not prove membership in a configured channel
- hidden commands must come from one of the configured `Channels`

Use this only when channel membership is part of the access policy. For normal user authorization, prefer `Users`, `RequireAdmin`, `Authorizer`, or elevation.

## User and Admin Policy

### Users

`Users` is an allow-list of canonical usernames.

```yaml
Users:
- alice
- bob
- ops-*
```

If `Users` is empty, all users are allowed by this setting. Entries are matched with filepath-style patterns, so `ops-*` can match usernames such as `ops-alice`.

### RequireAdmin

`RequireAdmin: true` restricts the whole plugin to robot administrators.

```yaml
RequireAdmin: true
```

Admins come from `AdminUsers` in `robot.yaml`, plus scheduled/automatic tasks controlled by robot configuration.

### AdminCommands

`AdminCommands` restricts individual commands to administrators.

```yaml
AdminCommands:
- clear-cache
- restart-service
```

Every command listed here must exist in `Commands` or `MessageMatchers`, or the plugin is disabled during config loading.

## Authorization

Authorization is for policy checks delegated to another plugin, usually for group or role decisions. It runs after admin checks and private-command checks.

### AuthorizedCommands

`AuthorizedCommands` lists commands that require authorization.

```yaml
AuthorizedCommands:
- deploy
- rollback
```

Every command listed here must exist in `Commands` or `MessageMatchers`, or the plugin is disabled.

### AuthorizeAllCommands

`AuthorizeAllCommands: true` requires authorization for every command in the plugin.

```yaml
AuthorizeAllCommands: true
```

### Authorizer

`Authorizer` overrides the robot-wide `DefaultAuthorizer` for this plugin.

```yaml
Authorizer: groups
```

The authorizer must be a plugin. When a command needs authorization, Gopherbot calls the authorizer's `_authorize` command with the protected plugin name, `AuthRequire`, the command name, and the command arguments.

If a plugin configures `Authorizer` but no commands actually require authorization, Gopherbot treats that as a configuration error.

### AuthRequire

`AuthRequire` is an optional string passed to the authorizer, commonly a group or role name.

```yaml
AuthRequire: production-deployers
```

The help system can also use an authorizer's `_usergroups` support to hide or show command help based on `AuthRequire`.

## Elevation

Elevation is an additional confirmation step, often MFA-like. It runs after authorization.

For human approval workflows, see [User Approval Elevation](../security/userapproval.md).

### ElevatedCommands

`ElevatedCommands` lists commands that require elevation.

```yaml
ElevatedCommands:
- deploy
- restart-service
```

After a successful elevation, the pipeline remains elevated for its lifetime.

### ElevateImmediateCommands

`ElevateImmediateCommands` lists commands that always require a fresh elevation prompt.

```yaml
ElevateImmediateCommands:
- rotate-secret
```

Use this sparingly for especially sensitive commands.

### Elevator

`Elevator` overrides the robot-wide `DefaultElevator` for this plugin.

```yaml
Elevator: totp
```

The elevator must be a plugin that implements the engine `_elevate` callback.

## Catch-All Plugins

### CatchAll

`CatchAll: true` lets a plugin handle unmatched directed commands.

```yaml
CatchAll: true
```

The plugin receives engine command `_catchall` and the full unmatched message text as the first argument.

There should usually be only one generic catch-all plugin in a location. Multiple catch-all matches can make routing ambiguous.

### CatchAllModes

`CatchAllModes` narrows which command-addressing modes a catch-all handles.

```yaml
CatchAllModes:
- name
- direct
```

Allowed values:

- `alias`: addressed through the robot alias
- `name`: addressed by robot name or mention
- `direct`: sent in a DM
- `hidden`: sent through hidden/ephemeral transport

Invalid values disable the plugin.

## Replies and Prompt Matching

### ReplyMatchers

`ReplyMatchers` are shared prompt/reply matchers used by plugin code that asks the user for a reply.

```yaml
ReplyMatchers:
- Label: confirm
  Regex: '(?i:y(?:es)?|no?)'
```

Fields:

- `Label`: label used by the plugin when waiting for a reply
- `Regex`: Go regex used to match the reply
- `ChannelOnly`: match only normal channel messages
- `Contexts`: optional context labels

`Label` values that start with a capital letter are reserved for stock reply matchers and disable the plugin.

## Plugin-Specific Config

### Config

`Config` is free-form YAML for the plugin itself.

```yaml
Config:
  DefaultRegion: us-east-1
  MaxItems: 20
```

Gopherbot stores this raw config and exposes it to the plugin through the Robot API. The engine validates the rest of the plugin file with known fields, but it intentionally does not validate the shape inside `Config`.

`Config` is the right place for plugin-owned options. Secrets should normally come from config template `secret` references or attached parameter sets, not hard-coded plaintext.

## Timeouts

### TimeOuts

`TimeOuts` overrides the robot-wide plugin timeout defaults for this plugin.

```yaml
TimeOuts:
  Warn: 2m
  Kill: 5m
```

Fields:

- `Warn`: how long the plugin can run before an operator warning
- `Kill`: how long the plugin can run before termination or manual-intervention alert

Durations are Go duration strings such as `30s`, `2m`, or `1h`. Integer values are accepted as nanoseconds. A value of `0` disables that threshold. If both thresholds are non-zero, `Kill` must be greater than `Warn`.

## Disabling a Plugin

### Disabled

`Disabled: true` disables the plugin.

```yaml
Disabled: true
```

You can disable a plugin in `robot.yaml` before plugin config is loaded, or in `conf/plugins/<plugin>.yaml` while loading plugin-specific config.

## Invalid or Legacy Keys

These keys are not valid in v3 plugin config:

- `CommandMatchers`: use `Commands`
- `Help`: use command-linked `Usage`, `Summary`, `Examples`, and `Keywords`
- `Helptext`: use `Summary` and `Examples`
- `Privileged`: set this in `robot.yaml`

These declaration fields belong in `robot.yaml`, not plugin YAML:

- `Path`
- `Description`
- `NameSpace`
- `ParameterSets`
- `Parameters`
- `Homed`

`NameSpace` and `ParameterSets` are explicitly ignored when they appear in plugin YAML. Put them in the plugin declaration in `robot.yaml`.

## Effective Top-Level Key Index

This index lists the top-level keys that have an effect in v3 plugin config.

| Key | Purpose |
| --- | --- |
| `AdminCommands` | Commands restricted to robot admins |
| `AllChannels` | Make the plugin available in all normal channels |
| `AllowedPrivateCommands` | Commands allowed in DMs or hidden/private contexts |
| `AmbientMatchCommand` | Let `MessageMatchers` also inspect directed commands |
| `AuthorizeAllCommands` | Require authorization for all plugin commands |
| `AuthorizedCommands` | Commands that require authorization |
| `Authorizer` | Plugin-specific authorizer override |
| `AuthRequire` | Group/role/policy string passed to the authorizer |
| `CatchAll` | Handle unmatched directed commands |
| `CatchAllModes` | Restrict catch-all handling by addressing mode |
| `Channel` | Channel used for scheduled/pipeline plugin contexts |
| `Channels` | Public channels where the plugin is visible |
| `Commands` | Directed command matchers |
| `Config` | Free-form plugin-owned configuration |
| `Disabled` | Disable the plugin |
| `ElevatedCommands` | Commands that require elevation |
| `ElevateImmediateCommands` | Commands that always require fresh elevation |
| `Elevator` | Plugin-specific elevator override |
| `MatchUnlisted` | Let ambient matchers run for unlisted users |
| `MessageMatchers` | Ambient message matchers |
| `ReplyMatchers` | Prompt/reply matchers |
| `RequireAdmin` | Restrict the whole plugin to admins |
| `RequireAllCommandsPrivate` | Require private context for every command |
| `RequiredPrivateCommands` | Commands that must run privately |
| `RestrictPrivateChannels` | Enforce channel restrictions for private commands |
| `TimeOuts` | Per-plugin timeout thresholds |
| `Users` | Canonical username allow-list |

## InputMatcher Field Index

`Commands`, `MessageMatchers`, and `ReplyMatchers` use the same matcher shape, with restrictions noted above.

| Field | Used By | Purpose |
| --- | --- | --- |
| `Command` | `Commands`, `MessageMatchers` | Command token passed to plugin |
| `Regex` | all matcher lists | Go regex matcher |
| `SimpleMatcher` | `Commands` only | Simplified directed-command matcher |
| `Usage` | `Commands`, `MessageMatchers` | Help usage text |
| `Summary` | `Commands`, `MessageMatchers` | Help summary |
| `Examples` | `Commands`, `MessageMatchers` | Help examples |
| `Keywords` | `Commands`, `MessageMatchers` | Help search terms |
| `Label` | `ReplyMatchers` | Reply matcher label |
| `ChannelOnly` | all matcher lists | Restrict match to non-thread channel messages |
| `Contexts` | all matcher lists | Labels for context memory based on captures |
