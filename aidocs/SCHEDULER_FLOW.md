# Scheduler Decisions

Scheduled and `@init` jobs are administrator-authored configuration, so they
start as `automaticTask=true` without inventing a user identity. This is why
they may pass admin gates; a future user-created schedule must use a different
authorization model.

Scheduling must converge on the ordinary pipeline runner. It must not bypass
task privilege, timeout, logging, failure, or parameter rules. The scheduler
owns timing only.

Configuration reload replaces the active schedule set after successful config
processing. Shutdown must prevent new scheduled work before waiting for active
pipelines.

Inspect `bot/scheduler.go`, `bot/tasks.go`, and `bot/run_pipelines.go` for the
current implementation and config fields.
