# Job Queues Design

Status: current v3 design and implementation notes for queue-triggered jobs.

This document describes a new job queue facility for Gopherbot. Queue providers
live under `queues/`, register through interfaces in `robot/`, and poll external
queue systems for text payloads that trigger configured jobs by UUID.

## Architecture Preflight Summary

Core architectural invariants:

- Startup must remain deterministic and traceable. Queue providers are optional
  runtime components and must start only after normal engine configuration,
  connector startup, job loading, scheduler setup, plugin initialization, and
  runtime git capture have completed.
- Shared authorization and business policy remain engine-owned. Queue providers
  only retrieve opaque queue bodies; they do not decide which job to run.
- Permission and identity policy remain username-authoritative for user-driven
  flows. Queue triggers are administrator-configured automatic job starts, not
  user messages.
- Connector isolation remains intact. Queue providers are not connectors and
  must not call `IncomingMessage` or set `DirectMessage`, `BotMessage`, or other
  message-context flags.
- Configuration precedence remains explicit: root `robot.yaml` selects enabled
  queue providers, and provider-specific settings live in `conf/queues/`.
- Secret access remains explicit and scope-based. UUID triggers may be loaded
  through `{{ secret "NAME" }}` in job config, but extensions do not get any API
  for listing queue providers, UUIDs, or queue configuration.

Startup ordering constraints:

- Existing startup order is preserved through pre-connect config load, brain
  init, module init, connector runtime init, post-connect config load, scheduler
  registration, plugin init, runtime git capture, and ready signaling.
- Queue provider startup is appended after that existing post-connect work.
  A queue item can only start a job after the engine has a complete current
  config and task list.
- Shutdown must stop queue providers before waiting for in-flight pipelines, so
  external queues stop introducing new work while the engine drains.

Connector assumptions:

- Connectors remain the only authority for chat message context and transport
  identity.
- Queue providers do not participate in multi-protocol inbound routing.
- Queue-started job output uses the job's configured channel and the engine's
  default outbound protocol behavior, matching scheduled jobs.

Message routing model:

- Queue bodies do not enter `handler.IncomingMessage` or `dispatch.go`.
- The queue path is a new job-start entrypoint that converges on
  `worker.startPipeline`, like scheduled jobs and current job triggers.
- Existing chat job triggers and `run job` behavior remain unchanged.

Identity model:

- Queue-triggered jobs run with `automaticTask=true`, the same trust stance used
  for scheduled jobs and current job triggers.
- Queue providers supply queue metadata, not users. The pipeline should expose
  queue origin metadata through dedicated `GOPHER_QUEUE_*` values rather than
  pretending there is a canonical chat username.

## Goals

- Add a first-class provider family for job queues, parallel to connectors,
  brains, and history providers.
- Support root `QueueProviders` configuration with provider settings in
  `conf/queues/<provider>.yaml`.
- Add optional per-job `UUIDTrigger` configuration.
- Start queue providers after the robot is fully initialized.
- Parse queue bodies of the form:

```text
<uuid> <shell-escaped arguments>
```

- Match only the first UUID-width bytes of the body against configured job
  UUIDs.
- Preserve job arguments with spaces by parsing shell-escaped argument text.
- Log an error for malformed or unknown UUID bodies, and log an info event when
  a job is triggered from a queue.
- Implement the first provider as `queues/gcloud`, backed by Google Pub/Sub.

## Non-Goals

- Do not route queue items through chat connectors.
- Do not add a public extension API for queue provider discovery or queue
  configuration.
- Do not add user-schedulable jobs or change user authorization semantics.
- Do not make queue acknowledgment depend on job success in the first slice.
  The first implementation treats the external queue item as accepted once the
  engine validates the body and starts the job pipeline.
- Do not guarantee global ordering for remote queue systems that do not provide
  it. Providers can choose conservative local concurrency defaults.

## Configuration Contract

Root `robot.yaml` gets an optional `QueueProviders` list:

```yaml
QueueProviders:
- gcloud
```

Provider-specific configuration lives under `conf/queues/`:

```yaml
# conf/queues/gcloud.yaml
QueueConfig:
  ProjectID: "my-gcp-project"
  SubscriptionID: "job-triggers-pull"
  CredentialsEncryptedFile: "gopherbot-key.json.enc"
  MaxOutstandingMessages: 1
  NumGoroutines: 1
```

Notes:

- `QueueProviders` follows normal config merge behavior. A custom list replaces
  the installed list unless the custom config uses `AppendQueueProviders`.
- The installed default robot should not enable any queue provider by default.
- `QueueConfig` is valid only in `conf/queues/<provider>.yaml`; it is invalid
  in root `robot.yaml`.
- Queue provider config files are templated and layered like other provider
  files, so custom robots can use `{{ secret "NAME" }}` and
  `{{ variable "NAME" }}` in provider config.

Jobs get an optional `UUIDTrigger`:

```yaml
# conf/jobs/myfirstjob.yaml
Channel: jobs
UUIDTrigger: {{ secret "MYFIRSTJOB_UUID" | printf "%q" }}
```

Validation rules:

- `UUIDTrigger` is optional.
- If set, it must be a canonical UUID string accepted by `github.com/google/uuid`.
- Enabled jobs must not share a `UUIDTrigger`; duplicate UUIDs fail config load.
- Disabled jobs do not register queue UUIDs.
- UUID values are normalized for matching but never logged.

## Provider Interfaces

`robot/queues.go` defines a queue-specific contract rather than widening the
base connector `robot.Handler` path.

Expected shape:

```go
type QueueMessage struct {
    ID         string
    Body       []byte
    Attributes map[string]string
}

type QueueDisposition int

const (
    QueueAck QueueDisposition = iota
    QueueRetry
)

type QueueProvider interface {
    Run(stop <-chan struct{})
}

type InitializedQueueProvider struct {
    Provider QueueProvider
}

type QueueProviderRegistration struct {
    Initialize func(QueueHandler, *log.Logger) (InitializedQueueProvider, error)
}

type QueueHandler interface {
    GetQueueConfig(interface{}) error
    HandleQueueMessage(QueueMessage) QueueDisposition
    ReadEncryptedFile(path string) ([]byte, error)
    Log(level LogLevel, msg string, args ...interface{})
    GetInstallPath() string
    GetConfigPath() string
    RaisePriv(reason string)
}

func RegisterQueueProvider(name string, initialize func(QueueHandler, *log.Logger) (InitializedQueueProvider, error))
```

Design points:

- Providers import `github.com/lnxjedi/gopherbot/robot`, not `bot`.
- The bot supplies a provider-scoped `queueHandler`, similar to
  `connectorHandler`, so `GetQueueConfig` reads only that provider's
  `QueueConfig`.
- `HandleQueueMessage` is the only way a provider submits work to the engine.
- `QueueDisposition` lets the engine distinguish messages that should be
  acknowledged from messages that should be retried during shutdown or transient
  engine unavailability.

## Engine Runtime

`bot/queue_runtime.go` owns provider lifecycle.

Responsibilities:

- Read `currentCfg.queueProviders`.
- Resolve `robot.GetQueueProviderRegistration(name)`.
- Initialize each configured provider with a provider-scoped handler.
- Start each provider in its own goroutine with an independent stop channel.
- Log provider startup, provider exit, and provider errors with a
  `[queue:<provider>]` prefix.
- Stop all queue providers during shutdown before waiting for pipelines.
- Preserve isolation: one provider init or run failure must not stop other
  queue providers or connectors.

Reload behavior:

- On successful config reload, reconcile queue provider runtimes after the new
  config and task list are installed.
- Removed providers are stopped.
- Added providers are initialized and started.
- Existing providers are restarted when their provider is still configured.
  This keeps the first implementation simple and avoids a broad provider
  `Reload` contract.
- Outstanding remote messages may redeliver across provider restart, depending
  on the provider backend.

Startup placement:

```text
run()
  -> go runBrain()
  -> restoreSubscriptions()
  -> startConnectorRuntimes()
  -> loadConfig(false)
  -> initializeRuntimeGitState()
  -> startQueueProviderRuntimes()
  -> Log("Robot is initialized and running")
  -> signalRobotInitialized()
```

This preserves the existing startup phases and makes the readiness signal mean
queue providers have at least been started or attempted.

Shutdown placement:

```text
stop()
  -> triggerPromptShutdownSignal()
  -> shutdownQueueProviderRuntimes()
  -> state.Wait()
  -> releaseBrainLock()
  -> brainQuit()
  -> shutdownConnectorRuntimes()
```

If a queue message arrives while `state.shuttingDown` is true, the engine returns
`QueueRetry` so the provider can leave it for redelivery.

## Queue Body Parsing

Queue bodies are byte-oriented.

Rules:

- `uuidLen` is 36 bytes.
- Bodies shorter than 36 bytes are invalid.
- Only bytes `[0:36]` are considered for job selection.
- The UUID prefix must parse as a UUID.
- If the body has more data, byte 36 must be a single ASCII space.
- Argument text begins at byte 37.
- Empty argument text means zero arguments.
- The argument parser tokenizes shell-escaped words without command execution,
  variable expansion, globbing, or file access.
- `github.com/anmitsu/go-shlex` is the preferred parser because it is a lexical
  splitter and already appears in the module graph. If it proves unsuitable,
  use `mvdan.cc/sh/v3/syntax` only in a restricted parse-only wrapper that
  rejects shell operators and substitutions.

Examples:

```text
1104df4c-feeb-43ab-8c85-83663288cea9 alpha two\ words
1104df4c-feeb-43ab-8c85-83663288cea9 'alpha beta' gamma
```

The existing `resources/gcloud/scripts/queue-job.sh` POC matches this contract
by using `printf "%q "` for each argument and sending the payload as
`text/plain`.

## Job Matching And Pipeline Start

The engine queue handler converges on:

```go
func triggerJobFromQueue(provider string, msg robot.QueueMessage) robot.QueueDisposition
```

Flow:

1. If the engine is shutting down, return `QueueRetry`.
2. Parse and validate the queue body.
3. Lookup the normalized UUID in the current enabled-job UUID map.
4. If no job matches, log `Error` and return `QueueAck`.
5. Parse shell-escaped arguments.
6. Validate supplied arguments against the job's `Arguments` matchers:
   - Too few arguments is an error.
   - Existing behavior for extra arguments is preserved.
   - Invalid argument values log `Error`.
   - Queue triggers never prompt for missing arguments.
7. Log `Info` that job `<name>` was triggered from queue provider `<provider>`.
8. Start the job pipeline with a new pipeline type, `queuedJob`.
9. Return `QueueAck` once the pipeline has been accepted for execution.

The worker should be built like a scheduled job worker:

- `automaticTask=true`
- `Protocol` set to `currentCfg.defaultProtocol`, falling back to primary
- `Channel` set to the job's configured channel
- `Incoming` set to an empty `robot.ConnectorMessage`
- no canonical user is invented

Queue-origin metadata should be exposed to the job as both environment values
and secure parameters, matching the existing `GOPHER_START_*` pattern:

```text
GOPHER_QUEUE_PROVIDER
GOPHER_QUEUE_MESSAGE_ID
```

Provider-specific metadata should not be blindly copied into the environment.
If a provider needs to expose safe metadata later, add an allowlisted contract.

`startPipeline` should get a `queuedJob` status branch so job-channel messages
are clear, for example:

```text
Starting queued job 'deploy alpha', run 14 - triggered by queue provider 'gcloud'
```

## Google Cloud Provider

The first provider lives in `queues/gcloud`.

Registration:

```go
func init() {
    robot.RegisterQueueProvider("gcloud", Initialize)
}
```

Config:

```go
type config struct {
    ProjectID                string
    SubscriptionID           string
    CredentialsEncryptedFile string
    MaxOutstandingMessages   int
    NumGoroutines            int
}
```

Behavior:

- Use `internal/gcloud.LoadServiceAccountCredentials` and
  `internal/gcloud.ServiceAccountClientOptions`.
- Resolve `ProjectID` from config or the service-account JSON.
- Normalize full subscription resource names down to subscription IDs, matching
  the Google Chat connector's `normalizeSubscriptionID` behavior.
- Default `CredentialsEncryptedFile` to `gopherbot-key.json.enc`.
- Default `SubscriptionID` to `job-triggers-pull`.
- Default `NumGoroutines` and `MaxOutstandingMessages` to `1`.
- Use `cloud.google.com/go/pubsub`.
- For each Pub/Sub message, call `HandleQueueMessage`.
- `QueueAck` maps to `msg.Ack()`.
- `QueueRetry` maps to `msg.Nack()`.
- The provider should not log body content.

Google resource alignment:

- `resources/gcloud/scripts/create-queue-resources.sh` creates topic
  `job-triggers` and subscription `job-triggers-pull`.
- `resources/gcloud/scripts/create-job-webhook-resources.sh` deploys a Cloud
  Function with a UUID-bearing URL and publisher service account.
- `resources/gcloud/scripts/webhook/index.js` publishes the raw request body.
- `resources/gcloud/scripts/queue-job.sh` sends the body contract expected by
  the engine.

## Security Notes

- Queue provider config is engine/provider config, not extension config.
- `UUIDTrigger` values are bearer trigger secrets and should normally be stored
  in `conf/variables/<environment>.yaml` under `Secrets`.
- Logs must never print configured UUIDs or full queue bodies.
- Unknown UUIDs should log only a short diagnostic such as provider name,
  provider message ID, and body length.
- Queue providers must not bypass `startPipeline`; privilege, timeout, logging,
  parameter, and failure behavior must remain centralized there.
- Queue-triggered jobs run as administrator-configured automatic tasks, like
  scheduled jobs. Do not derive admin status from provider metadata.
- Queue arguments are data. The engine must parse them without executing shell
  syntax or performing expansions.

## Acknowledgment Semantics

First-slice semantics are accepted-trigger acknowledgment:

- Ack after the engine validates the queue body and accepts the job pipeline.
- Ack malformed bodies and unknown UUIDs after logging an error.
- Retry only when the engine is shutting down or temporarily unable to make a
  routing decision.

Implications:

- A job returning failure does not cause queue redelivery.
- A robot crash after accepting a trigger may lose that remote queue item.
- Completion-coupled queue retries would require a broader design: pipeline
  completion callbacks, provider-specific ack-deadline management, retry/DLQ
  policy, and clear semantics for job failures.

## Impact Surface Report

### 1) Change Summary

- Slice name: job queue provider facility.
- Goal: allow remote queues to trigger configured jobs by UUID and arguments.
- Out of scope: completion-coupled queue retries, user-scheduled jobs, extension
  queue APIs, and connector-based queue routing.

### 2) Subsystems Affected

Expected implementation files and directories:

- `robot/queues.go`: queue provider registration and contracts.
- `queues/gcloud/`: Google Pub/Sub queue provider.
- `modules.go`: blank import for `queues/gcloud`.
- `bot/conf.go`: root `QueueProviders` parsing and queue provider config load.
- `bot/config_load.go`: provider config directory support for `QueueConfig`.
- `bot/config_validate.go`: `conf/queues/*.yaml` validation.
- `bot/tasks.go`: `Job.UUIDTrigger`.
- `bot/taskconf.go`: UUID trigger parsing, validation, duplicate detection.
- `bot/queue_runtime.go`: provider lifecycle and engine queue message handler.
- `bot/bot_process.go`: startup and shutdown placement.
- `bot/run_pipelines.go`, `bot/constants.go`, `bot/pipelinetype_string.go`:
  queued-job pipeline source.
- `conf/queues/gcloud.yaml`: installed default provider config.
- `conf/robot.yaml`, `robot.skel/conf/robot.yaml`, `conf/README.md`:
  examples and config layout.
- `aidocs/STARTUP_FLOW.md`, `aidocs/PIPELINE_LIFECYCLE.md`,
  `aidocs/COMPONENT_MAP.md`: canonical docs updates.
- `resources/gcloud/README.md`: queue resource and curl workflow docs.

Key functions and types:

- `loadConfig`, `loadProviderFileData`, `providerConfigDirectoryForKey`
- `loadTaskConfig`
- `worker.startPipeline`
- `run`
- `stop`
- new `startQueueProviderRuntimes`, `shutdownQueueProviderRuntimes`
- new `triggerJobFromQueue`

### 3) Current Behavior Anchors

- Startup/order anchors: `bot/bot_process.go` `initBot`, `run`, `stop`;
  `aidocs/STARTUP_FLOW.md`.
- Routing/message-flow anchors: `handler.IncomingMessage`,
  `worker.handleMessage`, `checkJobMatchersAndRun`,
  `runScheduledTask`, `worker.startPipeline`.
- Identity/authorization anchors: `handler.IncomingMessage` pre-pipeline user
  filters, `isAdminUser`, `jobSecurityCheck`, `automaticTask` behavior.
- Connector behavior anchors: `robot.Connector`, `connector_runtime.go`,
  connector-specific `Initialize` and `Run` methods.

### 4) Proposed Behavior

What changes:

- Root config can enable queue providers.
- Jobs can declare `UUIDTrigger`.
- Queue providers poll external systems and submit queue messages to the engine.
- The engine starts matching jobs from queue messages with a `queuedJob`
  pipeline type.

What does not change:

- Chat connector message routing.
- Existing job `Triggers`.
- Existing `run job` command behavior.
- Scheduled jobs.
- Extension Robot API behavior.
- Brain/history provider behavior.

### 5) Invariant Impact Check

- Startup determinism preserved: yes, queue provider startup is explicitly
  appended after post-connect runtime setup.
- Explicit control flow preserved: yes, queue messages enter a dedicated
  queue-runtime path and converge on `startPipeline`.
- Shared auth/policy remains in engine flows: yes, providers only fetch bodies.
- Permission checks remain username-based: yes for user-driven flows; queue
  triggers are explicit administrator configuration and use `automaticTask`.
- Connector ordering guarantees preserved: yes, queue providers do not touch
  connector routing.
- Config precedence still explicit: yes, root selection plus `conf/queues/`
  provider config.
- Multi-connector isolation preserved: yes, queue providers are independent
  runtime components and failures are provider-scoped.

### 6) Cross-Cutting Concerns

- Startup sequencing impact: new runtime start step after full config load.
- Config loading impact: new root key and provider config directory.
- Execution ordering impact: new job pipeline source; no changes to connector
  message ordering.
- Resource lifecycle impact: providers own remote clients and polling goroutines;
  engine owns stop channels and shutdown ordering.

### 7) Concurrency Risks

- Shared state touched: current config/task list and queue runtime registry.
- Locking assumptions: queue handler must snapshot current config under
  `currentCfg.RLock` before matching a UUID and starting a worker.
- Race risks: reload while queue item is being matched; provider shutdown while
  a message callback is active; duplicate queue delivery from provider restart.
- Mitigations: immutable config snapshots, provider stop channels, retry during
  shutdown, idempotent job design guidance, and unit tests for reload/shutdown
  behavior.

### 8) Backward Compatibility

- Existing robots are unaffected when `QueueProviders` is absent.
- Existing job configs are unaffected when `UUIDTrigger` is absent.
- Config schema changes are additive.
- No extension API signatures change.
- Operators enabling queues must add queue provider config and per-job UUIDs.

### 9) Validation Plan

Focused tests:

- Config validation accepts `conf/queues/gcloud.yaml` with `QueueConfig`.
- Root `QueueConfig` is rejected with a migration-style error.
- `QueueProviders` parses and normalizes provider names.
- `UUIDTrigger` accepts valid UUIDs and rejects invalid or duplicate values.
- Queue body parsing preserves arguments with spaces.
- Unknown UUID logs an error and returns `QueueAck`.
- Shutdown returns `QueueRetry`.
- Queue-triggered jobs start with `automaticTask=true`, expected args, and
  `GOPHER_QUEUE_*` metadata.

Broader regression tests:

- `go test ./bot ./robot ./queues/gcloud`
- process-backed integration suite for a fake/test queue provider or a
  queue-handler test hook
- `make` after core engine/runtime changes

Manual verification:

- In the Clu robot, configure `QueueProviders: [gcloud]`, add
  `UUIDTrigger: {{ secret "MYFIRSTJOB_UUID" | printf "%q" }}`, and start the
  robot.
- Use `resources/gcloud/scripts/queue-job.sh` with `WEBHOOK_URL`, `JOB_UUID`,
  and representative arguments including spaces.
- Confirm `robot.log` includes the info trigger log and the job channel shows
  the queued job start.

### 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md`: add queue provider startup/shutdown phase.
- `aidocs/PIPELINE_LIFECYCLE.md`: document queued-job entrypoint.
- `aidocs/SCHEDULER_FLOW.md`: no behavior change, but cross-reference queued
  jobs if helpful.
- `aidocs/COMPONENT_MAP.md`: add `queues/`, `conf/queues/`, and this document.
- `aidocs/V3_COMPATIBILITY_CONTRACT.md`: no required change unless config
  migration guidance changes.
- `conf/README.md`: add `queues/<provider>.yaml`.
- `resources/gcloud/README.md`: document queue resource scripts and curl flow.

### 11) Waiver

- None. This report is part of the design pass. Implementation should not begin
  until the design is accepted or explicitly revised.
