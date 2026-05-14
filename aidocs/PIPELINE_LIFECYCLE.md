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
- Unmatched directed-command location diagnostics next: when no command/message/job matched, the engine may emit a first-class "wrong location" response if the text regex-matches exactly one plugin command that is available to the same user in a different channel or private context. This runs before catch-alls and suppresses hints for command-level authorization when user visibility cannot be determined confidently (for example, authorizers without `usergroups` support).
- Catch‑alls (only when directly addressed and nothing matched): `bot/dispatch.go:handleMessage`.
- Thread subscriptions last (`Subscribe`/`Unsubscribe`) keyed by `protocol/channel/thread`, with legacy fallback for restored pre-protocol keys: `bot/dispatch.go:handleMessage`, `bot/subscribe_thread.go`.

Catch-all mode scoping:
- Plugins may optionally set `CatchAllModes` to any subset of `alias`, `name`, `direct`, `hidden`.
- `alias` means the robot was addressed through its alias prefix.
- `name` means the robot was addressed by name/mention form.
- `direct` means the command arrived in a DM context.
- `hidden` means the command arrived through hidden/ephemeral transport (`HiddenMessage=true`).
- During normal unmatched-command routing, dispatch only considers catch-all plugins whose `CatchAllModes` include the current `cmdMode`.
- Hidden unmatched commands first try a `hidden` mode catch-all, allowing robot owners to route hidden command recovery separately from DM recovery. If no hidden-specific catch-all matches, dispatch falls back to the normal `alias`/`name`/`direct` mode selection.
- Mode-scoped catchalls are treated as "specific" catchalls for precedence, so an alias-only recovery plugin can coexist with a name/direct AI fallback without colliding with generic fallback behavior.

## Private Command Policy (routing + safety guard)

- Private-command policy check runs at pipeline-start time: `bot/run_pipelines.go` calls `Robot.checkPrivateCommands` in `bot/allow_hidden.go`.
- Hidden/ephemeral transport support is still a connector capability (`robot.ConnectorCapabilities.HiddenCommands`) supplied by the initialized connector instance and consumed through `bot/connector_capabilities.go`.
- Connector registrations are static, but capability values are runtime/init-time so they can depend on protocol config (for example Slack slash-command enablement).
- A private command is allowed only if all applicable checks pass:
  - the command is listed in plugin `AllowedPrivateCommands`, listed in `RequiredPrivateCommands`, or covered by `RequireAllCommandsPrivate: true`
  - for hidden/ephemeral invocations, the message is explicitly addressed to this robot:
    - connector-marked bot message (`Incoming.BotMessage=true`, e.g. Slack slash route), or
    - name-addressed command mode (`cmdMode == "name"`).
  - if the plugin has `RestrictPrivateChannels: true`, the private context must satisfy the plugin channel restrictions.
- Practical effect: hidden `/...` payloads that are not bot-addressed by connector or name will not execute private commands.
- Plugins can require private invocation for selected commands with `RequiredPrivateCommands`, or for every command with `RequireAllCommandsPrivate`. The engine rejects non-private invocation with `This command is only available in a private context.` before plugin code runs.
- Some admin commands are private-required even though they are globally available by matcher location. For example, `dump robot`, `dump plugin`, `dump plugin default`, `list plugins`, `encrypt-secret`, and `generate-uuid` are implemented by `builtin-admin` and configured through `RequiredPrivateCommands`.
- Channel restrictions are primarily visibility, noise-control, and help-scoping policy:
  - public commands remain available only in configured `Channels`, unless `AllChannels: true`
  - private-capable commands are normally available from DMs and from hidden contexts in any channel where the robot is present, even when the plugin has configured `Channels`
  - help output may still use channel restrictions to decide what to advertise in a public channel, so a private-capable command can be runnable privately even when it is not advertised in that public channel's normal help
- `RestrictPrivateChannels: true` changes channel restrictions from visibility scoping into an access boundary for private-capable commands:
  - DMs are rejected when the plugin has restricted channels, because a DM does not prove membership in any allowed channel
  - hidden commands are allowed only when issued from one of the plugin's configured `Channels`
  - plugins without configured `Channels` should not use this setting as an access-control mechanism; use `Users`, admin commands, authorizers, or elevation for user authorization
- User-facing denial behavior is split cleanly:
  - if a hidden invocation uses a connector that does not support hidden/ephemeral transport, engine returns a single protocol-specific unsupported message
  - if the connector does support hidden transport but the user addressed it incorrectly, engine returns a single engine-authored guidance string built from the connector's concrete formatter (for example ``Use `/clu <command>` to address a private command.``)

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
- Directed `Commands` may now specify exactly one of:
  - `Regex` — raw Go regex, preserving legacy behavior
  - `SimpleMatcher` — simplified command syntax parsed into a first-class matcher object and compiled to exact-match regex data during config load (`bot/simple_matcher.go`)
- `CommandMatchers` and top-level `Help` are rejected in v3 plugin config validation.
- Intended `SimpleMatcher` semantics for directed commands:
  - case-insensitive by default
  - leading/trailing whitespace tolerated through the normal command compile wrapper
  - runs of whitespace are still collapsed during dispatch retry, preserving existing whitespace-forgiveness
  - spaces in the spec act as command separators and match either spaces or dashes in input
  - plain literal text is required and non-capturing
  - `/a|b|c/` is required non-capturing synonym text for choices the plugin does not need to know
  - `(label:a|b|c)` is a required labelled capturing choice; the selected value arrives as a positional plugin arg
  - `(:a|b|c)` is a required capturing choice with no diagnostic label
  - `[label:a|b|c]` is an optional labelled capturing choice/phrase; omitted values arrive as `""`
  - `[:a|b|c]` is an optional capturing choice with no diagnostic label
  - `{a|b|c}` is optional non-capturing noise text
  - typed captures use `<name:type>` or `<type>` and arrive positionally in the task handler
  - when a bracketed group contains a typed capture slot, the slot is the semantic capture; the wrapper should not create a second positional arg
  - bare `foo|bar` is intentionally not part of the grammar; use `/foo|bar/`, `(:foo|bar)`, `[:foo|bar]`, or `{foo|bar}`
  - a `SimpleMatcher` exact match starts the normal pipeline path; a unique syntax match may produce targeted feedback when the command skeleton matches exactly but one captured value is invalid
  - detailed authoring contract: `devdocs/SimpleMatcher.md`
  - diagnostic design: `aidocs/SIMPLE_MATCHER_DIAGNOSTICS.md`
- Ambient matchers continue to load from `MessageMatchers` and remain regex-only.
- Reply matchers and job argument matchers remain regex-based.

## Directed Command Syntax Diagnostics

- `SimpleMatcher` diagnostics are engine-owned command routing behavior, not connector behavior.
- Exact matches always win and preserve existing multiple-match safety behavior.
- A syntax diagnostic is eligible only when the command skeleton matches exactly and a single labelled choice or typed capture explains the failure.
- If the skeleton does not match exactly, dispatch continues to the normal unmatched-command path so help/fallback can suggest the best command.
- If multiple visible syntax diagnostics are possible, dispatch should avoid guessing and continue to normal unmatched-command handling.
- Syntax diagnostics must be considered only for commands visible to the current username and location; if command-level visibility is indeterminate, suppress the diagnostic.

## Pipeline Start (what gets called)

- Plugin match → `startPipeline(..., plugCommand|plugMessage, ...)`: `bot/dispatch.go:checkPluginMatchersAndRun`, `bot/constants.go` `pipelineType`.
- Job trigger / command → `startPipeline(..., jobTrigger|jobCommand, ...)`: `bot/jobrun.go:checkJobMatchersAndRun`, `bot/constants.go` `pipelineType`.
- Queue trigger → `triggerJobFromQueue(...)` parses the queue body, matches a
  job `UUIDTrigger`, and starts `startPipeline(..., queuedJob, ...)` without
  entering connector message routing: `bot/queue_runtime.go`.
- `startPipeline` now stamps each pipeline with `startedAt`, effective timeout settings, and operator-channel routing metadata before the primary task runs: `bot/run_pipelines.go`, `bot/pipecontext.go`.
- A bounded live log buffer is attached to every pipeline through `newPipelineLiveLogger(...)`: `bot/history.go`, `bot/pipeline_monitoring.go`.
  - The live buffer tees normal history logging and keeps recent section markers, `Robot.Log(...)` / `worker.Log(...)`, and child stdout/stderr even when a job/plugin later discards persisted history (`KeepLogs: 0`).

## Task Execution + Privilege Anchors

- Each task invocation uses `executeTask` -> `callTask` -> `go callTaskThread(...)` + return channel wait: `bot/task_execution.go`, `bot/calltask.go`.
- Compiled-in Go tasks/plugins remain trusted in-process engine code.
- File-backed extensions run through child process boundaries:
  - external executables use `pipeline-child-exec`
  - Lua, JavaScript, Gopherbot shell, and interpreted Go use `pipeline-child-rpc`
- When privilege separation is active, the parent sets `GOPHER_PRIVSEP_CHILD_ROLE`; the child commits to that role before interpreter or external script code starts.
- `startPipeline` sets pipeline privilege context (`pipeContext.privileged`) from the starter task: `bot/run_pipelines.go`.
- Adding privileged work to unprivileged pipelines is blocked in pipeline mutation APIs: `bot/robot_pipecmd.go`.
- Privsep supplementary-group policy is checked during startup before workload execution; see `aidocs/EXECUTION_SECURITY_MODEL.md`.

For a full execution/security walkthrough, see `aidocs/EXECUTION_SECURITY_MODEL.md`.

## Pipeline Monitoring And Timeouts

- Default warn/kill thresholds now live in `conf/robot.yaml` under:
  - `TimeOuts.Plugin.Warn`
  - `TimeOuts.Plugin.Kill`
  - `TimeOuts.Job.Warn`
  - `TimeOuts.Job.Kill`
- Per-plugin/per-job overrides live in `conf/plugins/<name>.yaml` and `conf/jobs/<name>.yaml` under `TimeOuts.Warn` / `TimeOuts.Kill`.
- Effective timeout resolution rules:
  - explicit task-level value overrides the type default
  - explicit `0` disables that threshold for the task
  - when both thresholds are non-zero, `Kill` must be greater than `Warn`
- A watchdog goroutine is started for active pipelines with any effective timeout: `bot/pipeline_monitoring.go`.
  - Warn threshold posts an operator-facing alert with WID, pipeline/task, start time, age, and a recent live-log excerpt.
  - Plugin alerts go to `DefaultJobChannel`; job alerts go to the job's configured channel.
  - Kill threshold appends a timeout marker to the live/history log and then:
    - cancels RPC-backed child work when available
    - kills external process groups for executable child work
    - emits a manual-intervention alert instead of force-killing compiled-in Go work
- Admin inspection commands for active pipelines:
  - `ps` is available only in direct/hidden message contexts because task arguments can contain sensitive operator data.
  - `ps` defaults to sectioned `Plugins` / `Jobs` output with pipeline `ID`, compact `AGE`, `USER`, pipeline name, current task, command/source, args, and an explicit hint for `ps -v`.
  - `ps -v` includes `STARTED`, execution class, OS child PID (`OSPID`), and parent pipeline (`FROM`) details.
  - `get-pipeline-log <id>` returns the current live buffer for an active pipeline.
- Admin secret helper commands:
  - `encrypt-secret <secret>` returns base64 ciphertext using the robot encryption key, matching `gopherbot encrypt <string>` output.
  - `generate-uuid` returns the same plaintext/encrypted UUID pair as `gopherbot uuid`.
  - Both commands are configured as private-required `builtin-admin` commands, so the engine rejects public-channel invocation before plugin code runs.
- Failure diagnostics now favor operator-facing alerts plus live-log excerpts over relying only on `<plugin>-fail.log`.
  - compiled-in Go panic recovery logs stack traces into the live buffer
  - interpreter/external stderr and traceback text is preserved in the same live buffer and alert path

## Pipeline Assembly (tasks/jobs/plugins)

- API surface: `robot/robot.go` methods `AddTask`, `AddJob`, `AddCommand`.
- Enforcement + mutation: `bot/robot_pipecmd.go` (e.g., `pipeTask`, `Robot.AddTask`).
- `AddJob` appends a job task to the current primary pipeline; when executed it runs as a child pipeline context (`bot/run_pipelines.go`, `worker.runPipeline` with `child.startPipeline(...)`).
- Child job pipelines started via `AddJob` do not inherit parent pipeline `SetParameter` state; pass required data as explicit job args or use the built-in `GOPHER_START_*` environment metadata exposed by `startPipeline`.
- Child job outbound protocol context is inherited from the parent pipeline context (not implicit default-protocol fallback), so command-origin protocol and `AddJob`-spawned status routing remain aligned.
- Queue-triggered jobs use the job's configured channel, the runtime default
  protocol, and `automaticTask=true`, like scheduled jobs. They expose
  `GOPHER_QUEUE_PROVIDER` and `GOPHER_QUEUE_MESSAGE_ID` as queue-origin
  metadata.
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
- Operational implication: jobs should not treat `AddCommand` as "resume as user" behavior. Reconnect/onboarding flows should instead use explicit send/prompt APIs such as `SendUserChannelMessage` and `PromptUserChannelForReply`, with any resume state carried explicitly in durable state files or parameters.
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
