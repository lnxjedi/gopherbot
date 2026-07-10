# Elevation Model

AI-onboarding view of Gopherbot elevation: intent, configuration surface,
engine control flow, extension API behavior, and current implementation notes.

## Conceptual Intent

Elevation is a second assurance check for sensitive interactive actions. It is
not the same as authorization.

- Authorization answers: "Is this canonical user allowed to perform this
  action?"
- Elevation answers: "How certain are we that this request really came from
  that user right now, rather than from an unlocked or compromised chat
  session?"

Common elevator implementations are MFA/2FA mechanisms such as TOTP or Duo.
The historical docs also allow broader designs such as prompting another user
to approve an action. This is why the implementation is plugin-based rather
than hard-coded to a single MFA provider.

Gopherbot's security model treats elevation as engine-owned policy. Connectors
own transport and identity normalization, but they must not decide whether a
command is elevated. Elevation decisions operate on the canonical username
already resolved by the connector/engine boundary.

## Canonical Names

In v3, the engine-owned callback command is `_elevate`.

Older external docs may say an elevator plugin is called with command
`elevate`. Treat that as legacy documentation. Current code and v3 migration
docs reserve leading-underscore callback names for engine lifecycle commands:
`_configure`, `_init`, `_authorize`, `_usergroups`, `_elevate`, `_catchall`,
`_subscribed`, and `_expiresub`.

Configured user-facing plugin commands and message matchers must not use the
leading-underscore namespace.

## Configuration Surface

Root robot config:

- `DefaultElevator`: default plugin name used for commands/jobs/tasks that need
  elevation and do not specify their own `Elevator`.

Task/job/plugin config:

- `Elevator`: override the root `DefaultElevator` for that task, job, or
  plugin.

Plugin-only config:

- `ElevatedCommands`: command names that require elevation but may reuse a
  previous successful elevation according to the elevator plugin's timeout
  policy.
- `ElevateImmediateCommands`: command names that require immediate elevation
  every time. The elevator receives `immediate=true` and should not rely on an
  existing timeout window.

Non-plugin jobs/tasks:

- A non-plugin task/job is considered elevation-required when it has an
  explicit `Elevator` configured.
- Interactive job execution also runs the job security check before starting
  the job pipeline.

Implementation note:

- Authorization misconfiguration has more config-load fail-fast validation than
  elevation. Elevation misconfiguration remains guarded at runtime by
  `bot/elevate.go`, which reports configuration or technical failures to the
  user and audit log.

## Engine Flow

Message-driven plugin pipelines enter through:

- connector -> `handler.IncomingMessage` in `bot/handler.go`
- dispatcher/matchers in `bot/dispatch.go`
- `startPipeline` in `bot/run_pipelines.go`

For each job or plugin task in a non-automatic pipeline, the security check
order in `run_pipelines.go` is:

1. Admin check
2. Required-private/private-command checks for plugins
3. Authorizer plugin
4. Elevator plugin

The order matters:

- Admin checks run before authorizers because admins bypass the authorizer.
- Elevation runs after base authorization because it is additional identity
  assurance, not the permission decision itself.
- Connectors do not participate in these checks.

Scheduled/init jobs set `automaticTask=true`, so normal user authorization and
elevation checks are skipped. This is intentional because scheduled jobs are
configured by administrators through robot configuration. Future user-scheduled
features must not reuse `automaticTask=true` without their own access-control
model.

## Pipeline State

`pipeContext.elevated` / `worker.elevated` is pipeline-lifetime state.

- The flag starts false for a pipeline.
- When required elevation succeeds, `bot/elevate.go` sets `w.elevated=true`.
- Once true, later tasks in the same pipeline do not re-challenge.
- Do not reset this flag mid-pipeline.
- If elevation fails at any point, the pipeline security check fails and the
  protected task does not run.

This pipeline-local flag is separate from elevator-provider timeout state. For
example, TOTP/Duo may remember that a canonical user elevated recently, but the
engine still records whether this specific pipeline has passed the elevation
gate.

## Elevator Callback Contract

`Robot.elevate(task, immediate)` in `bot/elevate.go` selects the elevator:

1. Use `task.Elevator` when set.
2. Otherwise use root `DefaultElevator`.
3. Fail with configuration error if no effective elevator exists.
4. Look up the named task and require it to be a plugin.
5. Call the plugin with command `_elevate` and one argument:
   - `"true"` when immediate elevation is required
   - `"false"` otherwise

Return-value contract:

- `robot.Success`: elevation succeeded.
- `robot.Fail`: elevation was evaluated and denied.
- `robot.MechanismFail`: a technical problem prevented a decision.
- `robot.ConfigurationError`: misconfiguration prevented a decision.
- `robot.Normal` or any other value is not success. `robot.Normal` is treated
  as a mechanism failure because auth/elevation plugins must explicitly return
  `robot.Success`.

User-facing failure behavior:

- Missing effective elevator: `Sorry, elevation failed due to a configuration error`
- Elevator mechanism failure: `Sorry, elevation failed due to a problem with the elevation service`
- Elevator explicit denial: `Sorry, this command requires elevation`

Elevator plugins may send additional explanatory messages before returning.

## `immediate` Semantics

`immediate=true` means the elevator should require fresh confirmation now.
This is used for `ElevateImmediateCommands` and for extension code that calls
`Elevate(true)`.

`immediate=false` means the elevator may apply its own timeout/cache policy.
The engine does not implement the timeout. The elevator plugin owns that
provider-specific policy.

Current shipped behavior:

- `builtin-totp` stores successful elevation timestamps by canonical username.
  It supports `TimeoutType: idle` and `TimeoutType: absolute`.
- Duo uses the same general timeout model, plus Duo preauth/auth flow and
  remembered device/method preferences.
- `builtin-userapproval` ignores timeout reuse and asks the requester to select
  one configured human approver for each elevation attempt.

## Extension API

Extensions can request elevation dynamically through:

- Go interface: `robot.Robot.Elevate(immediate bool) bool`
- Engine method: `bot.Robot.Elevate(immediate bool)`
- External JSON API function: `Elevate`
- Interpreter/RPC support for Lua, JavaScript, Gopherbot shell, Yaegi Go, and
  other HTTP-backed external script libraries

This exists for commands whose need for elevation depends on arguments or
runtime state, or for uncommon plugins that want to require elevation for every
command from inside plugin logic.

`Elevate(...)` uses the current task's effective elevator and the same
`_elevate` callback path as configured `ElevatedCommands`.

## Built-In Elevators

### `builtin-totp`

Registered in `bot/builtin_totp.go` as `builtin-totp`.

Behavior:

- `_init` loads `Config.Users` into an in-memory username-to-secret map.
- `check` lets a user validate a TOTP code manually.
- `_elevate <immediate>` prompts for a TOTP code when needed.
- It rejects reuse of the last valid TOTP code stored in the caller's memory
  namespace.
- Successful elevation updates the timeout map according to `TimeoutType`.

Important implementation detail:

- If a user has no configured TOTP secret, `checkOTP` returns
  `robot.MechanismFail`, which surfaces as a technical elevation failure.

### Duo

Registered by the optional Go plugin under `goplugins/duo`.

Behavior:

- `_elevate <immediate>` resolves the Duo username from canonical user data
  using `DuoUserString`.
- It performs Duo preauth, device/method prompting, Duo auth, and timeout
  caching.
- It can remember a user's preferred device/method in memory/datum state.
- The sample config lives in `conf/plugins/duo.yaml.sample` and is intended to
  be copied into a custom robot config when enabled.

### `builtin-userapproval`

Registered in `bot/builtin_userapproval.go` as `builtin-userapproval`.

Behavior:

- `_elevate <immediate>` loads `Config.FallbackApprovers` and
  `Config.PluginApprovers`. `Config.DefaultStrict` defaults to `true` when
  omitted.
- The current pipeline/plugin name comes from `r.pipeName`.
- `PluginApprovers.<pipeName>` supersedes `FallbackApprovers` when present.
- In strict mode, the requester is excluded from the eligible approver list, so
  users cannot approve their own elevation. A per-plugin `Strict` value
  overrides `DefaultStrict`.
- In non-strict mode, if the requester is listed in the effective approver list,
  the elevation is approved immediately without prompting another approver. If
  the requester is not listed, the normal requester-selection and approver DM
  flow is used.
- The requester receives a lowercase-letter menu identifying the action as
  `plugin/command` when a plugin command is known, such as `vpn/add-device`,
  and must reply with a single lowercase letter matched by the elevator's
  `approvalChoice` reply matcher.
- After a valid selection, the requester is told which approver is being asked
  before the robot sends that approver the DM prompt.
- The selected approver receives a DM yes/no prompt. `yes`/`y` approves and
  returns `robot.Success`; `no`/`n`, timeout, invalid requester choice, or
  missing eligible approvers fail elevation.
- The implementation supports at most 26 eligible approvers per effective list.

Config shape:

```yaml
ReplyMatchers:
- Label: approvalChoice
  Regex: '[a-z]'
Config:
  DefaultStrict: true
  FallbackApprovers: [ david, alice ]
  PluginApprovers:
    wireguard: [ alice, bob, david ]
    deploy:
      Approvers: [ alice, bob, david ]
      Strict: false
```

## Interactive Jobs

Interactive jobs started through the job command path run `jobSecurityCheck`
before the job pipeline starts.

Behavior:

- `automaticTask=true` skips the check.
- `RequireAdmin`, authorization, and elevation are checked against the job.
- If the check passes, the job pipeline starts.
- The started pipeline also follows normal job/plugin security checks when
  applicable.

Historical intent says interactive jobs should authorize/elevate and scheduled
jobs should not. The current code matches that broad intent.

## Testing Coverage

Current test coverage exercises:

- `ElevatedCommands` and `ElevateImmediateCommands` routing across Go,
  JavaScript, Lua, shell, and Python security suites.
- `Robot.Elevate(true)` API plumbing across runtimes.
- Failure handling when the TOTP elevator prompts but cannot validate because
  the test user lacks a configured secret.
- Successful `builtin-userapproval` requester-choice and selected-approver
  yes/no approval flow.
- Return-value handling that treats non-success as failure.

Known gap:

- There is not strong process-backed integration coverage for a successful
  TOTP elevation with a configured valid secret and subsequent timeout reuse.
  Add such coverage before changing provider timeout semantics or pipeline
  reuse behavior.

## Invariants For Future Changes

- Keep elevation decisions in engine-owned pipeline/security flow.
- Use canonical usernames for elevation identity and timeout keys.
- Do not derive elevation from connector flags, user message content, or
  transport-local IDs.
- Preserve check order: admin -> authorizer -> elevator.
- Preserve pipeline-lifetime `w.elevated` semantics.
- Preserve `robot.Success` as the only success return for elevator callbacks.
- Do not treat `robot.Normal` as successful elevation.
- If changing startup/config defaults for elevators, update
  `aidocs/STARTUP_FLOW.md`, root `UPGRADING-v3.md` when compatibility is
  affected, and default/template config under `conf/` or `robot.skel/`.
- If changing extension API behavior, update `aidocs/EXTENSION_API.md` and
  runtime checklist docs.
- If changing security ordering or failure behavior, update
  `aidocs/EXECUTION_SECURITY_MODEL.md` and add/adjust process-backed
  integration coverage.

## Useful Code References

- `bot/elevate.go`: effective elevator selection, `_elevate` call, return
  handling, event emission, `checkElevation`.
- `bot/run_pipelines.go`: security check order before job/plugin invocation.
- `bot/jobbuiltins.go`: interactive job security check.
- `bot/builtin_totp.go`: built-in TOTP elevator.
- `goplugins/duo/duo.go`: optional Duo elevator.
- `robot/robot.go`: public `Elevate(bool) bool` interface contract.
- `robot/robot_constants.go`: task return values and auth/elevation semantics.
- `conf/plugins/builtin-totp.yaml`: TOTP config shape.
- `conf/plugins/duo.yaml.sample`: Duo config shape.
- `integration/suites/data/Test*Security.yaml`: current elevation failure-path
  integration coverage.
