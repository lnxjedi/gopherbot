# Scheduler Flow (Cron → Pipeline)

AI‑onboarding view: config source, scheduler setup, and the exact pipeline entrypoints.

## Entry Points (call graph)

- Config load populates scheduled jobs: `bot/conf.go` type `ConfigLoader` field `ScheduledJobs`.
- Scheduler setup: `bot/scheduled_jobs.go` (func `scheduleTasks`).
- Scheduled/`@init` run path: `bot/scheduled_jobs.go` (func `runScheduledTask`) → `bot/run_pipelines.go` (method `startPipeline`).

## Config Source (what to inspect)

- Scheduled jobs are defined in `conf/robot.yaml` under `ScheduledJobs` and loaded into `ConfigLoader.ScheduledJobs` (`bot/conf.go` type `ConfigLoader`).
- Each scheduled entry is a `ScheduledTask` with a `Schedule` (cron spec) and a `TaskSpec` (`bot/tasks.go` types `ScheduledTask`, `TaskSpec`).

## Scheduler Setup (when and how)

- The scheduler is created in `scheduleTasks()` using `robfig/cron`: `bot/scheduled_jobs.go` (func `scheduleTasks`).
- The scheduler uses `currentCfg.timeZone` if set, otherwise system timezone.
- Jobs with `Schedule == "@init"` are not added to cron; they run once via `initJobs()` which is called from `loadConfig(false)` in `bot/conf.go`.

## Cron Tick → Pipeline

- For each scheduled entry, `scheduleTasks()` registers a cron function that calls `runScheduledTask()`: `bot/scheduled_jobs.go` (func `scheduleTasks`).
- `runScheduledTask()` builds a worker with `automaticTask=true` and calls `startPipeline()` with pipeline type `scheduled` (or `initJob` for `@init`): `bot/scheduled_jobs.go` (func `runScheduledTask`), `bot/constants.go` type `pipelineType`.
- `startPipeline()` sets up pipeline context and executes tasks: `bot/run_pipelines.go` (method `startPipeline`).

## Validation Gates (why a scheduled job won't run)

- Scheduled entries must reference a job; non‑job names are rejected with a log message: `bot/scheduled_jobs.go` (func `scheduleTasks`).
- Disabled jobs are skipped.
- Scheduled jobs must have a `Channel` set on the job/task.

## Fast Debug Pointers (AI use)

- If a schedule doesn't fire: check `bot/scheduled_jobs.go` (func `scheduleTasks`) for log lines and verify the cron spec in `conf/robot.yaml`.
- If a schedule fires but no pipeline runs: check the job name resolves to a job in `bot/scheduled_jobs.go` (func `scheduleTasks`) and `bot/run_pipelines.go` (func `startPipeline`).

## AI Checklist (verified entrypoints)

- Find scheduled job config: `conf/robot.yaml` `ScheduledJobs`.
- Confirm config load target: `bot/conf.go` type `ConfigLoader` field `ScheduledJobs`.
- Confirm scheduler setup: `bot/scheduled_jobs.go` (func `scheduleTasks`).
- Confirm scheduled run entrypoint: `bot/scheduled_jobs.go` (func `runScheduledTask`) → `bot/run_pipelines.go` (method `startPipeline`).
