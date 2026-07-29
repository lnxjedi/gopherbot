# Pipeline Decisions

Read `bot/handler.go`, `bot/dispatch.go`, and `bot/run_pipelines.go` for current
control flow. The durable routing order is:

1. Pre-pipeline identity/self/ignore/startup gates.
2. Existing prompt/reply waiters.
3. Directed plugin commands.
4. Ambient plugin matchers.
5. Job triggers and interactive job commands.
6. Conservative wrong-location diagnostics.
7. Catch-alls for addressed, unmatched input.
8. Thread subscriptions last.

Do not reorder without an explicit behavioral redesign. Engine callback command
names use the reserved leading-underscore namespace; extension-authored matcher
commands must not.

## Routing nuances

- Self messages may trigger jobs but not ordinary plugin/catch-all/subscription
  paths. This supports workflows that react to bot-authored transport events.
- `BotUser` accounts may issue explicit commands and trigger jobs, but do not
  participate in ambient/catch-all/subscription routing.
- Prompt waiters are scoped by protocol, canonical user, channel, and thread.
  Shutdown interrupts them immediately.
- Catch-all selection is mode-aware; ambiguous equally specific candidates do
  not run.
- Thread-scoped state includes protocol to prevent cross-protocol collisions.

## Private command model

Private transport is not authorization. The engine checks that a command is
private-capable/required, that hidden input addressed this robot, and then runs
normal authorization/elevation.

Plugin `Channels` normally scope public noise, routing, and help visibility;
private-capable commands remain usable in DMs or hidden contexts.
`RestrictPrivateChannels` is the explicit exception: it rejects DMs and permits
hidden invocation only from configured channels. It is still a location gate,
not user authorization.

## Pipeline state

- Starter plugin/job fixes pipeline privilege for its lifetime.
- Unprivileged pipelines cannot add privileged work.
- Successful elevation persists within the pipeline.
- Scheduled, init, and queue-triggered jobs are `automaticTask` because their
  creation is administrator-controlled; this is not a reusable model for
  user-scheduled jobs.
- Jobs preserve immutable `GOPHER_START_*` origin context even if later tasks
  change channel or working context.
- Queue providers and schedulers must converge on normal `startPipeline`
  execution so timeout, logging, privilege, and failure behavior stay shared.
