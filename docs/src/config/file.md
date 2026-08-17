# Configuration Overview

Gopherbot configuration is layered. The installed engine ships working defaults, and your robot repository supplies the small set of changes that make the robot yours.

The most important rule is:

- `robot.yaml` selects robot-wide behavior and declares extensions
- dedicated files hold detailed connector, provider, plugin, and job configuration
- custom files override shipped defaults

## File Layout

For a custom robot, configuration normally lives under `custom/conf/`:

```text
custom/conf/
  robot.yaml
  environments/
    development.yaml
    production.yaml
  protocols/
    ssh.yaml
    slack.yaml
  brains/
    file.yaml
  history/
    file.yaml
  queues/
    gcloud.yaml
  plugins/
    hello.yaml
  jobs/
    nightly-report.yaml
  variables/
    common.yaml
    production.yaml
```

## What Goes Where

Robot-wide selectors and policy live in `robot.yaml`.

Examples:

- `PrimaryProtocol`
- `DefaultProtocol`
- `SecondaryProtocols`
- `Brain`
- `HistoryProvider`
- `QueueProviders`
- `DefaultMessageFormat`
- `AdminUsers`
- `UserRoster`

See the [`robot.yaml` reference](robot-yaml.md) for the full list of top-level options.

Detailed config lives in dedicated files:

- connector config: `conf/protocols/<protocol>.yaml`
- brain config: `conf/brains/<provider>.yaml`
- history config: `conf/history/<provider>.yaml`
- queue config: `conf/queues/<provider>.yaml`
- plugin config: `conf/plugins/<plugin>.yaml`
- job config: `conf/jobs/<job>.yaml`

See the [plugin config reference](plugin-yaml.md) for `conf/plugins/<plugin>.yaml`.

This split keeps boundaries clean:

- transport concerns stay with connectors
- provider concerns stay with providers
- command and help metadata stays with the plugin that owns it
- robot-wide authorization and identity policy stays visible at the top level

## Loading Order

Gopherbot loads most configuration in two layers:

1. installed defaults from the engine tree
2. custom overrides from the robot home

For example, when Gopherbot loads `conf/plugins/ping.yaml`, it first loads the installed `conf/plugins/ping.yaml`, then overlays `custom/conf/plugins/ping.yaml` when that custom file exists.

This same pattern applies to `robot.yaml`, protocols, providers, plugins, and jobs.

## Merge Behavior

- maps merge recursively
- scalar values override
- arrays replace by default
- arrays append only when the custom key uses the `Append*` prefix

That merge model is why some connector-local identity data uses lists instead of maps: lists replace cleanly, which prevents accidental inheritance of unwanted defaults.

## Appending to Arrays

By default, a custom array replaces the shipped default array:

```yaml
UserRoster:
- UserName: alice
  Email: alice@example.com
```

If you want to append to an existing array instead, prefix the key with `Append`:

```yaml
AppendUserRoster:
- UserName: bob
  Email: bob@example.com
```

During merge, Gopherbot strips the `Append` prefix and appends the new array entries to the existing `UserRoster`.

This works for array-valued keys throughout the config tree, such as:

- `AppendUserRoster`
- `AppendAdminUsers`
- `AppendCommands`
- `AppendAllowedPrivateCommands`
- `AppendScheduledJobs`

Use `Append*` deliberately. Replacing arrays is often safer for security-sensitive lists because it prevents accidentally inheriting default users, channels, commands, or provider entries.

Implementation detail that matters for troubleshooting: `Append` only changes merge behavior when the existing value and the new value are both arrays. For maps, nested maps merge recursively whether or not the key starts with `Append`; for scalar values, the custom value replaces the default.

## Template Expansion

Config files are Go text templates before they are parsed as YAML. This applies to `robot.yaml`, included environment files, protocol files, provider files, plugin files, and job files.

See [Config Templates](templates.md) for the helper reference, examples, and secret/variable guidance.

## Example

```yaml
{{ $environment := env "GOPHER_ENVIRONMENT" | default "development" }}
{{ printf "environments/%s.yaml" $environment | .Include }}
```

That is the standard v3 pattern for selecting environment-specific defaults in a scaffolded robot.
