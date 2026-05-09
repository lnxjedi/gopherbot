# Execution And Security Model (Current)

This document describes how pipeline execution and privilege separation currently work in the engine, with concrete code anchors.

## Scope

- Message/job-triggered pipeline execution model.
- Per-task execution threading model.
- Current privilege-separation behavior (setuid-nobody startup plus one-shot child role commitment for file-backed extensions).

## High-Level Flow

1. Connector submits `ConnectorMessage` to `handler.IncomingMessage` (`bot/handler.go`).
2. Engine creates a `worker` and starts `go w.handleMessage()` (`bot/handler.go`).
3. Matcher routing eventually calls `w.startPipeline(...)` (`bot/dispatch.go`, `bot/run_pipelines.go`).
4. Pipeline tasks are run via `w.runPipeline(...)` -> `w.executeTask(...)` -> `w.callTask(...)` (`bot/run_pipelines.go`, `bot/task_execution.go`, `bot/calltask.go`).
5. `callTask` runs each task in `go w.callTaskThread(...)` and waits on a return channel (`bot/calltask.go`).

## Pipeline Concurrency Semantics

- A pipeline is represented by one `worker` + `pipeContext` (`bot/pipecontext.go`).
- Pipelines run concurrently with each other (message handlers each run in their own goroutines).
- Tasks within a single pipeline are sequenced by `runPipeline` (exclusive queueing can defer some tasks), but each task body executes in a dedicated task goroutine via `callTask`.
- Global counters/waiting for shutdown are tracked with `state.pipelinesRunning` + `state.WaitGroup` (`bot/run_pipelines.go`, `bot/bot_process.go`).

## Execution Boundary

- `runPipeline` delegates task invocation through `worker.executeTask(...)` (`bot/task_execution.go`).
- Explicit invariant: `taskGo` tasks (compiled-in handlers implemented in `bot/*`) remain trusted in-process engine code.
- Current routing by task class:
  - `taskGo` -> in-process `callTask`.
  - interpreter-backed external:
    - `.lua` -> child RPC process (`gopherbot pipeline-child-rpc`) via parent-managed `lua_run` / `robot_call`.
    - `.js` -> child RPC process (`gopherbot pipeline-child-rpc`) via parent-managed `js_run` / `robot_call`.
    - `.gsh` -> child RPC process (`gopherbot pipeline-child-rpc`) via parent-managed `gsh_run` / `gsh_get_config` / `robot_call`.
    - `.go` -> child RPC process (`gopherbot pipeline-child-rpc`) via parent-managed `go_plugin_run` / `go_job_run` / `go_task_run` / `robot_call`.
  - external executable (non-interpreter path) -> child process runner (`gopherbot pipeline-child-exec`) via `callTask` options.
  - external executable plugin default-config (`configure`) -> child process runner from `getDefCfgThread`.
  - external Lua plugin default-config -> child RPC process via `lua_get_config`.
  - external JavaScript plugin default-config -> child RPC process via `js_get_config`.
  - external Gopherbot shell plugin default-config -> child RPC process via `gsh_get_config`.
  - external Go plugin default-config -> child RPC process via `go_get_config`.

When privilege separation is active, the parent supplies `GOPHER_PRIVSEP_CHILD_ROLE` to `pipeline-child-exec` and `pipeline-child-rpc`. `Start(...)` commits the child to that role before any interpreter or external executable code runs.

## Privilege Separation Bootstrap

On supported Unix platforms (`bot/privsep.go` for Linux/BSD and `bot/privsep_darwin.go` for macOS), privilege separation is initialized in `init()`:

- If `uid != euid`, engine treats:
  - `privUID = uid` (invoking user)
  - `unprivUID = euid` (setuid account, commonly `nobody`)
- It also records `privGID` and `unprivGID`.
- Startup swaps the parent engine back to the invoking effective UID/GID while preserving the setuid/setgid unprivileged saved IDs for later child commits.
- `privSep` is enabled only when this initialization succeeds.

Runtime visibility is logged through `checkprivsep()` in startup (`bot/start.go`).

After pre-connect config load, startup validates the unprivileged child role with `privsep-self-check`:

- the self-check child commits to the unprivileged role
- the child reports UID/GID/supplementary groups as JSON
- startup fails closed if UID/GID are wrong or retained supplementary groups are outside `PrivsepAllowAllSupplementaryGroups` / `PrivsepAllowedSupplementaryGroups`

Installed `conf/robot.yaml` defaults to `PrivsepAllowAllSupplementaryGroups: false` and `PrivsepAllowedSupplementaryGroups: []`.

## Child Role Commitment

File-backed extension children have one role:

- `privileged`: permanently commits to the invoking robot user UID/GID
- `unprivileged`: permanently commits to the setuid/setgid unprivileged UID/GID

The parent chooses the role from engine policy. The child must not decide whether privileged execution is allowed.

Platform mechanics differ:

- Linux/BSD use the saved setuid/setgid state to commit the child directly to the selected real/effective IDs.
- macOS uses the Darwin-compatible two-step for the unprivileged role (`seteuid`/`setegid`, then `setreuid`/`setregid`) before extension code starts.

Legacy thread-scoped helpers (`raiseThreadPriv`, `raiseThreadPrivExternal`, `dropThreadPriv`) remain for parent-owned privileged operations and migration compatibility, but normal file-backed extension execution is process-oriented.

## Task-Type Execution Behavior

- Compiled-in Go tasks/plugins:
  - handler runs in-process as trusted engine code
  - compiled-in extensions are not treated as unprivileged sandboxed code
- External interpreted tasks (`.go` via yaegi, `.lua`, `.js`, `.gsh`) now execute in child RPC processes.
  - The child commits to the parent-selected privsep role before the RPC loop starts.
  - Parent keeps policy/routing/identity authority and services Robot API calls over RPC.
  - For `.gsh`, shell utilities such as `ls`, `grep`, `jq`, `mktemp`, and `tar` stay inside the child process; only Robot operations cross back to the parent engine over RPC.
  - Parent tracks active RPC child process in `worker.osCmd` and request-cancel hook in `worker.rpcCancel`.
  - RPC request lifecycle now uses bounded handshake/request/shutdown/child-exit waits with explicit error classes.
- External executable tasks:
  - parent starts an internal child runner process (`gopherbot pipeline-child-exec`) with separate process group (`Setpgid: true`).
  - The child commits to the parent-selected privsep role before execing the target script/interpreter.
  - child runner executes exactly one external command, streams stdout/stderr, exits with command status.
  - parent tracks child pid in `worker.osCmd` for admin `ps`/`kill` and timeout watchdog kill handling.

`getDefCfgThread` (plugin configure/default-config path) routes file-backed configure calls through the same child process boundary.

## Operator Observability And Kill Scope

- Active pipelines now keep:
  - `startedAt`
  - effective warn/kill timeout profile
  - operator-channel routing target
  - a bounded live log ring buffer for recent pipeline output
- The live buffer is engine-owned and exists independently of persisted history retention.
  - It captures section markers, engine log lines (`Robot.Log(...)` / `worker.Log(...)`), and child stdout/stderr.
- Admin/operator inspection surface:
  - `ps` is available only in direct/hidden message contexts because task arguments can contain sensitive operator data.
  - `ps` is the default low-risk view for active pipelines and intentionally omits OS PID.
  - `ps` presents the kill/log handle as pipeline `ID`, groups active plugins and jobs into separate sections, and includes a compact age plus a `ps -v` hint.
  - `ps -v` exposes OS child PID (`OSPID`) plus start time, parent pipeline, and execution-class details for operators who need kill/debug context.
  - `get-pipeline-log <id>` exposes the current live ring buffer for an active pipeline.
- Timeout watchdog kill scope is intentionally narrower than alert scope:
  - external executable child pipelines can be killed by process group
  - RPC-backed interpreter/child-Go pipelines can be canceled and/or killed through parent-held child state
  - compiled-in Go plugins/jobs/tasks are not force-killed in v2.9; the engine emits a manual-intervention alert instead
- Practical implication: timeout monitoring is broad, but hard termination only applies where the parent process actually owns a killable child boundary.

## Engine-Level Security Controls Around Pipelines

- Pipeline privilege starts from starter task (`Plugin.Privileged` or `Job.Privileged`) in `startPipeline` (`bot/run_pipelines.go`).
- Adding privileged tasks/plugins/jobs to unprivileged pipelines is blocked (`bot/robot_pipecmd.go`).
- Some pipeline parameters/secrets are gated by privilege checks in environment assembly (`bot/run_pipelines.go` comments + logic around inherited params).

## Extension Secret Access Boundary

Gopherbot treats extension secret access as explicit and scope-based.

- An extension may receive secrets when the robot administrator explicitly assigns them to that extension through task/plugin configuration, `ParameterSets`, or task config retrieved through `GetTaskConfig()`.
- An extension may also access secrets it previously stored inside brain/memory state owned by its own namespace.
- Unprivileged robot methods must not expose shared/global secret-bearing configuration, nor provide discovery of secrets assigned to other extensions.
- The `conf/variables/*.yaml` `Secrets` registry is a configuration-template
  input only. It is resolved during config load through `{{ secret "NAME" }}`;
  there is no runtime Robot API for listing or reading global variables/secrets.

Practical rule for engine APIs:

- `GetTaskConfig()` is acceptable for extension-specific secrets because the robot owner explicitly attached that config to the calling extension.
- Identity provider access must follow the same explicit-scoping rule: an extension may call identity credential/link methods only for providers whose credential `ParameterSet` is also attached to that extension.
- Generic robot methods must not return provider registries, parameter-set contents, or other broad config objects that could disclose secrets to untrusted plugins.

## User Permission Model Invariants

These apply to `bot/handler.go`, `bot/available.go`, `bot/authorize.go`, `bot/elevate.go`, and `bot/run_pipelines.go`.

### Pre-pipeline filters

- `IgnoreUsers` and `IgnoreUnlistedUsers` are checked in `handler.IncomingMessage` before any worker is created. They must remain pre-pipeline filters. Never move this logic into dispatch or pipeline code.
- The `IgnoreUsers` check is case-insensitive. New pre-pipeline user filtering must use the same comparison.
- `IgnoreUnlistedUsers` is now validation-aware: the engine only trusts a canonical inbound username for policy when the connector set `ConnectorMessage.ValidatedUser=true`.
- The one intentional exception is the user-validation OTP receive path: an exact 7-digit DM/hidden message is checked before normal user dropping so an unvalidated user can prove a protocol account to an administrator.

### Admin authority sources

- Admin status (`isAdminUser` in `bot/available.go`) has exactly two legitimate sources: the `adminUsers` config list (username match), or `w.automaticTask == true`. It must never be derived from user input, message content, connector-provided flags, or any runtime state modifiable by users.
- `automaticTask == true` grants admin unconditionally. This is intentional: cron jobs are scheduled by administrators through robot configuration. If a future user-schedulable ("at-job") feature is added, it must **not** use `automaticTask = true` — it requires its own access control model.

### Check ordering in `run_pipelines.go`

- Order is: **admin check → authorizer plugin → elevator plugin**. Admin check runs first because admins bypass the authorizer; elevation runs last because it is an additional confirmation step after base authorization is established.
- The `w.elevated` flag persists for the lifetime of the pipeline. Once elevated, subsequent tasks in the same pipeline do not re-challenge. Do not reset `w.elevated` mid-pipeline.

### Access control defaults

- `Task.Users` is a whitelist: an empty list means all users are permitted. Never invert this — empty must never restrict access.
- An authorizer plugin returning `robot.Normal` (0) is a mechanism failure, not success. Auth plugins must explicitly return `robot.Success` (1). Do not change this behavior.

### Private inspection/admin commands

- Private-command allow/deny remains engine-owned through plugin `AllowedPrivateCommands`, `RequiredPrivateCommands`, or `RequireAllCommandsPrivate`.
- Hidden/ephemeral invocations also require connector support and robot addressing; direct messages do not require hidden transport support.
- The broadened private admin surface does not weaken underlying auth/elevation checks:
  - `builtin-admin` exposes selected admin commands as private-capable through explicit command lists.
  - Legacy `builtin-dmadmin` inspection commands (`dump robot`, `dump plugin`, `dump plugin default`, `list plugins`) now live on `builtin-admin` as globally available private-required commands; public channel invocation is rejected by the engine before plugin code returns configuration data.
  - `builtin-history` and `builtin-jobcmd` can expose their allowed private commands.
  - job/history security checks still authorize against the target job/task and preserve normal admin/authorization/elevation ordering.
- Practical implication: private `ps`, `get-pipeline-log`, config inspection, `jobs`, and history lookups are a transport/privacy convenience, not a policy bypass.

## Message Context and Privacy Invariants

The concern is not command visibility (hard to hide) but message routing confidentiality: the bot accidentally broadcasting sensitive data to a channel, or treating a public channel message as if it were private.

### Connector authority over message context

- Connectors are the sole authority for `Incoming.DirectMessage` and `Incoming.HiddenMessage`. These flags must be set accurately by the connector and must not be modified by the engine or plugins after `handler.IncomingMessage` returns.
- Private-command requirements are enforced in engine pipeline startup before plugin code runs.

### Response routing — no implicit privatization

- `r.Say()` and `r.Reply()` reply in the same channel/DM context as the triggering message. The engine does not implicitly privatize responses. This must not change.
- Plugins or tasks that return sensitive data (credentials, tokens, personal info, secrets) must either:
  - Be configured through `RequiredPrivateCommands` / `RequireAllCommandsPrivate`, **or**
  - Explicitly call `r.Direct().Reply()` / `r.Direct().Say()` to force a DM response.
- Bot-initiated messages (not in response to a user command) containing per-user sensitive data must use `SendUserMessage` (DM path), not `SendChannelMessage`. There is no engine guard for this — it is a code review requirement.

## Practical Limitations (Current)

- This is not yet a strict multi-process sandbox model for all task types.
- Compiled-in tasks (`taskGo`, `bot/*`) still execute in the engine process.
- Lua, JavaScript, Gopherbot shell, external Go interpreter-backed tasks, and external executable tasks gain process isolation via parent/child execution.
- Cancellation semantics for long-running interpreter tasks are now available through admin `kill` and timeout watchdogs, but fine-grained task-level cancellation (beyond process termination) remains future work.
- Privsep does not drop supplementary groups on platforms that cannot do so without root; startup fails closed unless retained groups are explicitly allowed.
- Compiled-in Go extensions are trusted engine code and are not supported as unprivileged sandboxed extensions.
- Privilege separation is implemented on Linux/BSD (`bot/privsep.go`) and macOS/Darwin (`bot/privsep_darwin.go`).
- Other platforms use `bot/privsep_unsupported.go`: `privSep` remains false, privilege-switch helpers are no-ops, and startup logs that privilege separation is not available on that platform.
