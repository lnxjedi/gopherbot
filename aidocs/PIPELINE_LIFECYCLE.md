# Pipeline Lifecycle (Incoming Message)

AI‑onboarding view: entrypoints, decision points, and data flow for message‑driven pipelines, with concrete code refs.

## Entry Points (call graph)

- Connector → `handler.IncomingMessage` (normalizes user/channel/message, spawns worker): `bot/handler.go` (method `IncomingMessage` on type `handler`).
- Worker → `handleMessage` (message routing, matcher evaluation): `bot/dispatch.go` (method `handleMessage` on type `*worker`).
- Match → pipeline start via `startPipeline`: `bot/dispatch.go` (method `checkPluginMatchersAndRun`), `bot/run_pipelines.go` (method `startPipeline`).

## Key Data Structures (what to inspect)

- Worker context: `bot/handler.go` type `worker` (fields `msg`, `fmsg`, `isCommand`, `cmdMode`, `Incoming`, `Channel`, `User`).
- Matcher definitions: `bot/tasks.go` type `Plugin` fields:
  - directed command matchers in `Commands`
  - ambient matchers in `MessageMatchers`
  - both use `InputMatcher` metadata (`Usage`, `Summary`, `Examples`, `Keywords`) from the same file.
- Pipeline type enum: `bot/constants.go` type `pipelineType` (`plugCommand`, `plugMessage`, `catchAll`, `jobCommand`, etc.).

## Decision Points (routing order)

- Reply waiters first (prompt/reply): `bot/dispatch.go:handleMessage`.
- Direct commands → `Commands`: `bot/dispatch.go:handleMessage`, `bot/dispatch.go:checkPluginMatchersAndRun`.
- Ambient messages → `MessageMatchers`: `bot/dispatch.go:handleMessage`, `bot/dispatch.go:checkPluginMatchersAndRun`.
- Job triggers / `run job`: `bot/dispatch.go:handleMessage`, `bot/jobrun.go:checkJobMatchersAndRun`.
- Unmatched directed-command location diagnostics next: when no command/message/job matched, the engine may emit a first-class "wrong location" response if the text regex-matches exactly one plugin command that is available to the same user in a different channel or only via DM. This runs before catch-alls and suppresses hints for command-level authorization when user visibility cannot be determined confidently (for example, authorizers without `usergroups` support).
- Catch‑alls (only when directly addressed and nothing matched): `bot/dispatch.go:handleMessage`.
- Thread subscriptions last (`Subscribe`/`Unsubscribe`) keyed by `protocol/channel/thread`, with legacy fallback for restored pre-protocol keys: `bot/dispatch.go:handleMessage`, `bot/subscribe_thread.go`.

Catch-all mode scoping:
- Plugins may optionally set `CatchAllModes` to any subset of `alias`, `name`, `direct`.
- `alias` means the robot was addressed through its alias prefix.
- `name` means the robot was addressed by name/mention form.
- `direct` means the command arrived in a DM context.
- During unmatched-command routing, dispatch only considers catch-all plugins whose `CatchAllModes` include the current `cmdMode`.
- Mode-scoped catchalls are treated as "specific" catchalls for precedence, so an alias-only recovery plugin can coexist with a name/direct AI fallback without colliding with generic fallback behavior.

## Hidden Command Policy (routing + safety guard)

- Hidden-command policy check runs at pipeline-start time: `bot/run_pipelines.go` calls `Robot.checkHiddenCommands` in `bot/allow_hidden.go`.
- A hidden command is allowed only if both are true:
  - the command is listed in plugin `AllowedHiddenCommands`
  - the hidden message is explicitly addressed to this robot:
    - connector-marked bot message (`Incoming.BotMessage=true`, e.g. Slack slash route), or
    - name-addressed command mode (`cmdMode == "name"`).
- Practical effect: hidden `/...` payloads that are not bot-addressed by connector or name will not execute hidden commands.

## Self-Message Routing Nuance (HearSelf-style flows)

- `ConnectorMessage.SelfMessage=true` is treated specially.
- Normal plugin paths (`Commands`, `MessageMatchers`, catch-alls, thread subscriptions) are gated behind `!w.Incoming.SelfMessage` checks in `bot/dispatch.go:handleMessage`.
- Job triggers are checked first in `bot/jobrun.go:checkJobMatchersAndRun`, before the self-message early return:
  - This enables a pattern where a robot emits a formatted message and then reacts to that same message via a trigger job (for example, to capture a started thread ID from `GOPHER_START_THREAD_ID`).
- Practical implication: if you need plugin `MessageMatchers` to react, do not mark the inbound event as `SelfMessage=true`; if you need trigger-job reaction, `SelfMessage=true` is compatible.

## Prompt Waiter Lifecycle (Prompt* APIs)

- Prompt waiters are keyed by `protocol/user/channel/thread` and checked before command/message matcher routing: `bot/dispatch.go:handleMessage`, `bot/replyprompt.go`.
- Default prompt timeout is `45s`.
- Extended prompt timeout is `42m` only when both are true:
  - Incoming protocol is `ssh` or `terminal`.
  - Current task is compiled Go or interpreter-backed (`.go`, `.lua`, `.js`).
- During shutdown, in-progress prompt waits are interrupted immediately (returning `Interrupted`) instead of waiting for timeout, so long prompt windows do not block shutdown completion.

## Config → Matcher Data (where matchers come from)

- YAML source: `conf/plugins/*.yaml` (example `conf/plugins/ping.yaml`).
- Directed command matcher key: `Commands`.
- `CommandMatchers` and top-level `Help` are rejected in v3 plugin config validation.
- Ambient matchers continue to load from `MessageMatchers`.

## Pipeline Start (what gets called)

- Plugin match → `startPipeline(..., plugCommand|plugMessage, ...)`: `bot/dispatch.go:checkPluginMatchersAndRun`, `bot/constants.go` `pipelineType`.
- Job trigger / command → `startPipeline(..., jobTrigger|jobCommand, ...)`: `bot/jobrun.go:checkJobMatchersAndRun`, `bot/constants.go` `pipelineType`.

## Task Execution + Privilege Anchors

- Each task invocation uses `executeTask` -> `callTask` -> `go callTaskThread(...)` + return channel wait: `bot/task_execution.go`, `bot/calltask.go`.
- Privilege separation primitives are in `bot/privsep.go` (`dropThreadPriv`, `raiseThreadPriv`, `raiseThreadPrivExternal`) and rely on thread pinning (`runtime.LockOSThread`).
- `startPipeline` sets pipeline privilege context (`pipeContext.privileged`) from the starter task: `bot/run_pipelines.go`.
- Adding privileged work to unprivileged pipelines is blocked in pipeline mutation APIs: `bot/robot_pipecmd.go`.

For a full execution/security walkthrough, see `aidocs/EXECUTION_SECURITY_MODEL.md`.

## Pipeline Assembly (tasks/jobs/plugins)

- API surface: `robot/robot.go` methods `AddTask`, `AddJob`, `AddCommand`.
- Enforcement + mutation: `bot/robot_pipecmd.go` (e.g., `pipeTask`, `Robot.AddTask`).
- `AddJob` appends a job task to the current primary pipeline; when executed it runs as a child pipeline context (`bot/run_pipelines.go`, `worker.runPipeline` with `child.startPipeline(...)`).
- Child job pipelines started via `AddJob` do not inherit parent pipeline `SetParameter` state; pass required data as explicit job args or use the built-in `GOPHER_START_*` environment metadata exposed by `startPipeline`.
- Child job outbound protocol context is inherited from the parent pipeline context (not implicit default-protocol fallback), so command-origin protocol and `AddJob`-spawned status routing remain aligned.
- Tail-pipeline APIs: `robot/robot.go` methods `FinalTask`, `FailTask`, `FinalCommand`, `FailCommand`.
- Runtime stage ordering: primary tasks run first, then `Final*` tasks always run, and `Fail*` tasks run only when primary pipeline status is non-normal (`bot/run_pipelines.go`, `worker.startPipeline` + `worker.runPipeline`).
- `FinalTask` ordering is LIFO/FILO by design (cleanup stack behavior): `bot/robot_pipecmd.go`, `pipeTask` (`flavorFinal` prepends to `w.finalTasks`).
- `FailTask` ordering is append/in-order (FIFO): `bot/robot_pipecmd.go`, `pipeTask` (`flavorFail` appends to `w.failTasks`).
- Operational pattern: pair acquisition/setup in `AddTask` with cleanup in `FinalTask` (for example, `ssh-agent deploy` auto-registers `FinalTask("ssh-agent", "stop")` and `ssh-git-helper` host-key setup auto-registers `FinalTask("ssh-git-helper", "delete")`).
- `AddCommand` composes plugin work into the current pipeline; it does not inject a transport/user-originated inbound message.
- `AddCommand` only succeeds when:
  - it runs during the primary task stage (`primaryTasks`)
  - the provided command text matches the target plugin's `Commands`
- Built-in admin git flows (`update`, `switch-branch`, `default-branch`) call `AddCommand("builtin-admin", "reload")`; reload success/failure is mirrored to `GOPHER_START_*` origin context when available, while detailed pipeline output can still go to the job channel.
- Operational implication: jobs should not treat `AddCommand` as "resume as user" behavior. For reconnect/onboarding flows, prefer explicit user prompts/instructions and let the user invoke the next command.
- TODO (long-term): document and evaluate whether a dedicated user-scoped resume/injection primitive is needed, distinct from pipeline composition APIs.

## Fast Debug Pointers (AI use)

- If a message never starts a pipeline, trace `bot/handler.go` (func `IncomingMessage`) → `bot/dispatch.go` (func `handleMessage`) and verify matcher config in `conf/plugins/*.yaml`.
- If a pipeline starts but tasks don't run, inspect `bot/run_pipelines.go` (func `startPipeline`) and `bot/robot_pipecmd.go` (AddTask/AddJob/AddCommand validation).

## AI Checklist (verified entrypoints)

- Locate the message entrypoint: `bot/handler.go` (method `IncomingMessage`).
- Confirm routing order: `bot/dispatch.go` (method `handleMessage`).
- Confirm matcher definitions: `bot/tasks.go` type `Plugin` fields `Commands`, `MessageMatchers`.
- Confirm matcher config source: `conf/plugins/*.yaml` (example `conf/plugins/ping.yaml`).
- Confirm pipeline start: `bot/dispatch.go` (method `checkPluginMatchersAndRun`) → `bot/run_pipelines.go` (method `startPipeline`).
