# Pipeline Lifecycle (Incoming Message)

AI‑onboarding view: entrypoints, decision points, and data flow for message‑driven pipelines, with concrete code refs.

## Entry Points (call graph)

- Connector → `handler.IncomingMessage` (normalizes user/channel/message, spawns worker): `bot/handler.go` (method `IncomingMessage` on type `handler`).
- Worker → `handleMessage` (message routing, matcher evaluation): `bot/dispatch.go` (method `handleMessage` on type `*worker`).
- Match → pipeline start via `startPipeline`: `bot/dispatch.go` (method `checkPluginMatchersAndRun`), `bot/run_pipelines.go` (method `startPipeline`).

## Key Data Structures (what to inspect)

- Worker context: `bot/handler.go` type `worker` (fields `msg`, `fmsg`, `isCommand`, `cmdMode`, `Incoming`, `Channel`, `User`).
- Matcher definitions: `bot/tasks.go` type `Plugin` fields `CommandMatchers`, `MessageMatchers` (type `InputMatcher` in same file).
- Pipeline type enum: `bot/constants.go` type `pipelineType` (`plugCommand`, `plugMessage`, `catchAll`, `jobCommand`, etc.).

## Decision Points (routing order)

- Reply waiters first (prompt/reply): `bot/dispatch.go:handleMessage`.
- Direct commands → `CommandMatchers`: `bot/dispatch.go:handleMessage`, `bot/dispatch.go:checkPluginMatchersAndRun`.
- Ambient messages → `MessageMatchers`: `bot/dispatch.go:handleMessage`, `bot/dispatch.go:checkPluginMatchersAndRun`.
- Job triggers / `run job`: `bot/dispatch.go:handleMessage`, `bot/jobrun.go:checkJobMatchersAndRun`.
- Catch‑alls (only when directly addressed and nothing matched): `bot/dispatch.go:handleMessage`.

## Prompt Waiter Lifecycle (Prompt* APIs)

- Prompt waiters are keyed by `protocol/user/channel/thread` and checked before command/message matcher routing: `bot/dispatch.go:handleMessage`, `bot/replyprompt.go`.
- Default prompt timeout is `45s`.
- Extended prompt timeout is `42m` only when both are true:
  - Incoming protocol is `ssh` or `terminal`.
  - Current task is compiled Go or interpreter-backed (`.go`, `.lua`, `.js`).
- During shutdown, in-progress prompt waits are interrupted immediately (returning `Interrupted`) instead of waiting for timeout, so long prompt windows do not block shutdown completion.

## Config → Matcher Data (where matchers come from)

- YAML source: `conf/plugins/*.yaml` (example `conf/plugins/ping.yaml` `CommandMatchers`).
- Loaded into `Plugin.CommandMatchers` / `Plugin.MessageMatchers`: `bot/tasks.go` type `Plugin`.
- Populated during config load: `bot/taskconf.go` cases `CommandMatchers`, `MessageMatchers`.

## Pipeline Start (what gets called)

- Plugin match → `startPipeline(..., plugCommand|plugMessage, ...)`: `bot/dispatch.go:checkPluginMatchersAndRun`, `bot/constants.go` `pipelineType`.
- Job trigger / command → `startPipeline(..., jobTrigger|jobCommand, ...)`: `bot/jobrun.go:checkJobMatchersAndRun`, `bot/constants.go` `pipelineType`.

## Pipeline Assembly (tasks/jobs/plugins)

- API surface: `robot/robot.go` methods `AddTask`, `AddJob`, `AddCommand`.
- Enforcement + mutation: `bot/robot_pipecmd.go` (e.g., `pipeTask`, `Robot.AddTask`).

## Fast Debug Pointers (AI use)

- If a message never starts a pipeline, trace `bot/handler.go` (func `IncomingMessage`) → `bot/dispatch.go` (func `handleMessage`) and verify matcher config in `conf/plugins/*.yaml`.
- If a pipeline starts but tasks don't run, inspect `bot/run_pipelines.go` (func `startPipeline`) and `bot/robot_pipecmd.go` (AddTask/AddJob/AddCommand validation).

## AI Checklist (verified entrypoints)

- Locate the message entrypoint: `bot/handler.go` (method `IncomingMessage`).
- Confirm routing order: `bot/dispatch.go` (method `handleMessage`).
- Confirm matcher definitions: `bot/tasks.go` type `Plugin` fields `CommandMatchers`, `MessageMatchers`.
- Confirm matcher config source: `conf/plugins/*.yaml` (example `conf/plugins/ping.yaml`).
- Confirm pipeline start: `bot/dispatch.go` (method `checkPluginMatchersAndRun`) → `bot/run_pipelines.go` (method `startPipeline`).
