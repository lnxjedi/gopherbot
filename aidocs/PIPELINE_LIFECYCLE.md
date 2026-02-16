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
  - both use `InputMatcher` metadata (`Usage`, `Summary`, `Examples`, `Keywords`, `Helptext`) from the same file.
- Pipeline type enum: `bot/constants.go` type `pipelineType` (`plugCommand`, `plugMessage`, `catchAll`, `jobCommand`, etc.).

## Decision Points (routing order)

- Reply waiters first (prompt/reply): `bot/dispatch.go:handleMessage`.
- Direct commands → `Commands`: `bot/dispatch.go:handleMessage`, `bot/dispatch.go:checkPluginMatchersAndRun`.
- Ambient messages → `MessageMatchers`: `bot/dispatch.go:handleMessage`, `bot/dispatch.go:checkPluginMatchersAndRun`.
- Job triggers / `run job`: `bot/dispatch.go:handleMessage`, `bot/jobrun.go:checkJobMatchersAndRun`.
- Catch‑alls (only when directly addressed and nothing matched): `bot/dispatch.go:handleMessage`.
- Thread subscriptions last (`Subscribe`/`Unsubscribe`) keyed by `protocol/channel/thread`, with legacy fallback for restored pre-protocol keys: `bot/dispatch.go:handleMessage`, `bot/subscribe_thread.go`.

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
- `AddCommand` composes plugin work into the current pipeline; it does not inject a transport/user-originated inbound message.
- `AddCommand` only succeeds when:
  - it runs during the primary task stage (`primaryTasks`)
  - the provided command text matches the target plugin's `Commands`
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
