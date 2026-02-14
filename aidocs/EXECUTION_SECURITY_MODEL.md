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

## Execution Boundary (Slice 5 State)

- `runPipeline` delegates task invocation through `worker.executeTask(...)` (`bot/task_execution.go`).
- Explicit invariant for the multiprocess epic: `taskGo` tasks (compiled-in handlers implemented in `bot/*`) remain in-process.
- Current routing by task class:
  - `taskGo` -> in-process `callTask`.
  - interpreter-backed external:
    - `.lua` -> child RPC process (`gopherbot pipeline-child-rpc`) via parent-managed `lua_run` / `robot_call`.
    - `.go` / `.js` -> in-process `callTask` (not migrated yet).
  - external executable (non-interpreter path) -> child process runner (`gopherbot pipeline-child-exec`) via `callTask` options.
  - external executable plugin default-config (`configure`) -> child process runner from `getDefCfgThread`.
  - external Lua plugin default-config -> child RPC process via `lua_get_config`.

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
- External interpreted tasks (`.go` via yaegi, `.lua`, `.js`):
  - `.lua` now executes in a child RPC process; parent keeps policy/routing/identity authority and services Robot API calls over RPC.
  - `.go` via yaegi and `.js` remain in-process and still use thread-scoped privilege operations.
- External executable tasks:
  - parent path still applies privilege drop/raise before starting execution.
  - parent starts an internal child runner process (`gopherbot pipeline-child-exec`) with separate process group (`Setpgid: true`).
  - child runner executes exactly one external command, streams stdout/stderr, exits with command status.
  - parent tracks child pid in `worker.osCmd` for admin `ps`/`kill`.

`getDefCfgThread` (plugin configure/default-config path) also drops privilege before external configure calls and now routes external executable configure through `pipeline-child-exec` (`bot/calltask.go`).

## Engine-Level Security Controls Around Pipelines

- Pipeline privilege starts from starter task (`Plugin.Privileged` or `Job.Privileged`) in `startPipeline` (`bot/run_pipelines.go`).
- Adding privileged tasks/plugins/jobs to unprivileged pipelines is blocked (`bot/robot_pipecmd.go`).
- Some pipeline parameters/secrets are gated by privilege checks in environment assembly (`bot/run_pipelines.go` comments + logic around inherited params).

## Practical Limitations (Current)

- This is not yet a strict multi-process sandbox model for all task types.
- Compiled-in tasks and some interpreter-backed tasks (`.go`/`.js`) still execute in the engine process.
- Lua interpreter-backed tasks now gain process isolation via parent/child RPC.
- Long-lived correctness depends on careful `LockOSThread` usage and goroutine/thread lifecycle.

TODO (verify): if non-Unix builds are targeted in future, document the explicit fallback behavior when `bot/privsep.go` is excluded by build tags.
