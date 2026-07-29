# Job Queue Decisions

Queue providers are engine-owned runtime components, separate from chat
connectors. Providers fetch and acknowledge transport messages; the engine
parses, selects a configured job, and converges on normal pipeline execution.

- Providers are isolated from each other and from connectors.
- Queue configuration is provider-scoped and never exposed through the
  extension API.
- A job's UUID trigger is a bearer secret. Never log trigger UUIDs or full
  queue bodies.
- Queue arguments are data. Parsing may split quoted words but must never
  execute shell syntax, expand variables/globs, or read files.
- Matching queue jobs are `automaticTask` because both provider and job trigger
  are administrator configuration. Provider metadata must never create user or
  admin identity.
- UUID/timestamp deduplication is short-lived idempotency, not authorization or
  freshness validation.

Acknowledgment is intentionally coupled to engine acceptance, not job success:
malformed/unknown/duplicate messages are acknowledged after safe logging;
shutdown/transient inability to route may request retry; a later pipeline
failure does not cause queue redelivery. Completion-coupled retry would require
a separate design for callbacks, deadlines, retries, and dead-letter policy.

Queue providers start after full configuration and first plugin init, stop
before pipeline drain, and are restarted on reload. Inspect
`robot/queues.go`, `bot/queue_runtime.go`, and provider tests for current wire
formats and configuration fields.
