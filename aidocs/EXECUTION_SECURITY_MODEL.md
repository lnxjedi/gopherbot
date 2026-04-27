# Execution And Security Model (Current)

This document describes how pipeline execution and privilege separation currently work in the engine, with concrete code anchors.

## Scope

- Message/job-triggered pipeline execution model.
- Per-task execution threading model.
- Current privilege-separation behavior (`setreuid` + thread pinning).

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

## Execution Boundary (Slice 8 State)

- `runPipeline` delegates task invocation through `worker.executeTask(...)` (`bot/task_execution.go`).
- Explicit invariant for the multiprocess epic: `taskGo` tasks (compiled-in handlers implemented in `bot/*`) remain in-process.
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

## Privilege Separation Bootstrap

On supported Unix platforms (`bot/privsep.go` build tag: linux/bsd), privilege separation is initialized in `init()`:

- If `uid != euid`, engine treats:
  - `privUID = uid` (invoking user)
  - `unprivUID = euid` (setuid account, commonly `nobody`)
- Startup calls `syscall.Setreuid(unprivUID, privUID)` to initialize startup threads.
- `privSep` is enabled only when this initialization succeeds.

Runtime visibility is logged through `checkprivsep()` in startup (`bot/start.go`).

## Thread-Scoped Privilege Switching

Privilege changes are intentionally scoped to the current OS thread:

- `dropThreadPriv(reason)`:
  - `runtime.LockOSThread()`
  - `setReuid(unprivUID, unprivUID)`
- `raiseThreadPriv(reason)`:
  - ensure effective uid is `privUID` for current thread.
- `raiseThreadPrivExternal(reason)`:
  - `runtime.LockOSThread()`
  - `setReuid(privUID, privUID)` permanently for that locked thread.

Key invariant in current model: dropping/raising privilege for task execution relies on locked thread lifetime, not process isolation.

## Task-Type Execution Behavior

`callTaskThread` (`bot/calltask.go`) applies privilege operations before task execution:

- Compiled-in Go tasks/plugins:
  - `raiseThreadPriv` if pipeline/task is privileged, else `dropThreadPriv`.
  - Handler runs in-process.
- External interpreted tasks (`.go` via yaegi, `.lua`, `.js`, `.gsh`) now execute in child RPC processes.
  - Parent keeps policy/routing/identity authority and services Robot API calls over RPC.
  - For `.gsh`, shell utilities such as `ls`, `grep`, `jq`, `mktemp`, and `tar` stay inside the child process; only Robot operations cross back to the parent engine over RPC.
  - Parent tracks active RPC child process in `worker.osCmd` and request-cancel hook in `worker.rpcCancel`.
  - RPC request lifecycle now uses bounded handshake/request/shutdown/child-exit waits with explicit error classes.
- External executable tasks:
  - parent path still applies privilege drop/raise before starting execution.
  - parent starts an internal child runner process (`gopherbot pipeline-child-exec`) with separate process group (`Setpgid: true`).
  - child runner executes exactly one external command, streams stdout/stderr, exits with command status.
  - parent tracks child pid in `worker.osCmd` for admin `ps`/`kill` and timeout watchdog kill handling.

`getDefCfgThread` (plugin configure/default-config path) also drops privilege before external configure calls and now routes external executable configure through `pipeline-child-exec` (`bot/calltask.go`).

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

### Hidden inspection/admin commands

- Hidden-command allow/deny remains engine-owned and still requires connector support plus plugin `AllowedHiddenCommands`.
- The broadened hidden admin surface does not weaken underlying auth/elevation checks:
  - `builtin-admin` may expose most admin commands as hidden-capable, but `quit`, `restart`, and `abort` remain excluded.
  - Legacy `builtin-dmadmin` inspection commands (`dump robot`, `dump plugin`, `dump plugin default`, `list plugins`) now live on `builtin-admin` as globally available hidden-required commands; direct or channel-visible invocation is rejected by the handler before returning configuration data.
  - `builtin-history` and `builtin-jobcmd` can now expose their allowed hidden commands.
  - job/history security checks still authorize against the target job/task and preserve normal admin/authorization/elevation ordering.
- Practical implication: hidden `ps`, `get-pipeline-log`, config inspection, `jobs`, and history lookups are a transport/privacy convenience, not a policy bypass.

## Message Context and Privacy Invariants

The concern is not command visibility (hard to hide) but message routing confidentiality: the bot accidentally broadcasting sensitive data to a channel, or treating a public channel message as if it were private.

### Connector authority over message context

- Connectors are the sole authority for `Incoming.DirectMessage`. This flag must be set accurately by the connector and must not be modified by the engine or plugins after `handler.IncomingMessage` returns.
- `DirectOnly: true` on a task is enforced in `pluginAvailable` before the pipeline starts — the task will not match in a channel. This enforcement must not be weakened.

### Response routing — no implicit privatization

- `r.Say()` and `r.Reply()` reply in the same channel/DM context as the triggering message. The engine does not implicitly privatize responses. This must not change.
- Plugins or tasks that return sensitive data (credentials, tokens, personal info, secrets) must either:
  - Be marked `DirectOnly: true` (command can only be invoked via DM), **or**
  - Explicitly call `r.Direct().Reply()` / `r.Direct().Say()` to force a DM response.
- Bot-initiated messages (not in response to a user command) containing per-user sensitive data must use `SendUserMessage` (DM path), not `SendChannelMessage`. There is no engine guard for this — it is a code review requirement.

## Practical Limitations (Current)

- This is not yet a strict multi-process sandbox model for all task types.
- Compiled-in tasks (`taskGo`, `bot/*`) still execute in the engine process.
- Lua, JavaScript, Gopherbot shell, and external Go interpreter-backed tasks now gain process isolation via parent/child RPC.
- Cancellation semantics for long-running interpreter tasks are now available through admin `kill` and timeout watchdogs, but fine-grained task-level cancellation (beyond process termination) remains future work.
- Long-lived correctness depends on careful `LockOSThread` usage and goroutine/thread lifecycle.
- The setreuid/thread-pinned privilege separation implementation is compiled only on Linux, DragonFly BSD, FreeBSD, NetBSD, and OpenBSD (`bot/privsep.go`).
- Other platforms, including macOS/Darwin, use `bot/privsep_unsupported.go`: `privSep` remains false, privilege-switch helpers are no-ops, and startup logs that privilege separation is not available on that platform.
