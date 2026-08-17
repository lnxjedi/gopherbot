# User Approval Elevation

`builtin-userapproval` is an elevation plugin for commands that should require a human approval step instead of a personal MFA challenge. It is useful when a sensitive action should be approved by another trusted operator, such as a deploy, account change, firewall update, or credential rotation.

Use it when the question is:

- "Is this action important enough that another person should approve it?"

Do not use it as a substitute for base authorization. Authorization still decides whether the requester is allowed to ask for the action. Elevation adds a second confirmation step after that authorization succeeds.

## How the Flow Works

When a command requires elevation and uses `builtin-userapproval`:

1. Gopherbot finds the effective approver list for the current plugin.
2. In strict mode, the requester is removed from that list.
3. The requester chooses one remaining approver from a short lettered menu.
4. The selected approver receives a direct yes/no prompt.
5. If the approver replies `yes` or `y`, the protected command runs.

If the selected approver replies `no`, does not reply before the prompt times out, or there are no eligible approvers, elevation fails and the protected command does not run.

## Strict Mode

Strict mode prevents self-approval. It is enabled by default.

With strict mode enabled, this configuration:

```yaml
Config:
  FallbackApprovers: [ alice, bob ]
```

means:

- if `alice` requests elevation, only `bob` can approve
- if `bob` requests elevation, only `alice` can approve
- if `alice` is the only listed approver, Alice cannot approve herself and elevation fails

This is the safest default for most production robots.

## Non-Strict Mode

Non-strict mode allows a listed approver to approve their own elevation automatically. This can be useful for small teams, break-glass commands, personal robots, development environments, or actions where the approver list is being used as a lightweight operator allow-list.

To disable strict mode globally for this elevator:

```yaml
Config:
  DefaultStrict: false
  FallbackApprovers: [ alice, bob ]
```

With that configuration, if `alice` requests an elevated command and `alice` is in the effective approver list, Gopherbot approves the elevation immediately.

If the requester is not in the effective approver list, the normal choose-an-approver flow still runs.

## Configuration

Enable the elevator by setting it as the robot default:

```yaml
DefaultElevator: builtin-userapproval
```

Then configure the built-in plugin in `conf/plugins/builtin-userapproval.yaml`:

```yaml
ReplyMatchers:
- Label: approvalChoice
  Regex: '[a-z]'

Config:
  DefaultStrict: true
  FallbackApprovers: [ david ]
  PluginApprovers:
    deploy:
      Approvers: [ alice, bob, david ]
      Strict: false
    wireguard: [ alice, bob ]
```

`FallbackApprovers` is used when the current plugin is not listed under `PluginApprovers`.

`PluginApprovers` can use either form:

- list form: `wireguard: [ alice, bob ]`
- object form with a strict-mode override:

```yaml
PluginApprovers:
  deploy:
    Approvers: [ alice, bob, david ]
    Strict: false
```

The list form uses `DefaultStrict`. The object form can override strict mode for that plugin.

## Protecting Commands

In the target plugin's config, list the commands that require elevation:

```yaml
ElevatedCommands:
- deploy
- rollback

Commands:
- Command: deploy
  SimpleMatcher: "deploy <service:ident>"
- Command: rollback
  SimpleMatcher: "rollback <service:ident>"
```

These are ordinary directed command matchers; see the [SimpleMatcher reference](../reference/SimpleMatcher.md) for matcher syntax details.

If the robot has `DefaultElevator: builtin-userapproval`, those commands use user approval automatically.

To use this elevator for one plugin only, set `Elevator` in that plugin config:

```yaml
Elevator: builtin-userapproval
ElevatedCommands:
- rotate-key
```

## Choosing Approvers

Approvers are canonical Gopherbot usernames, not Slack IDs, email addresses, SSH usernames, or connector-local IDs. The connector maps transport identity to the canonical username before authorization and elevation run.

Use names from `UserRoster` and `AdminUsers`, such as:

```yaml
UserRoster:
- UserName: alice
  UserID: U123
- UserName: bob
  UserID: U456
```

Approver names are normalized by trimming whitespace and lowercasing before comparison.

## Operational Notes

Keep approver lists small and intentional. The requester sees a single-letter menu, and the implementation supports up to 26 eligible approvers.

Prefer strict mode for production operations where a second person should really be involved. Use non-strict mode only when self-approval is an intentional policy choice.

For commands that return sensitive data, combine elevation with private-command policy. User approval controls whether the command may run; it does not automatically move the response into a direct message.
