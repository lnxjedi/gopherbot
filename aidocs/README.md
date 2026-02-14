Gopherbot is an extensible automation framework designed as a persistent, Go-based chatbot ("robot"). It connects to chat platforms (or the terminal) and executes automation "pipelines" triggered by chat messages, scheduled events, or internal calls. Pipelines consist of compiled Go, interpreted scripts (Lua, JavaScript, dynamic Go), or external executables. A key v3 goal is self-containment, minimizing external tool dependencies. The robot features a pluggable "brain" for persistence and a flexible, template-based configuration system.

## What "Robot" Means (Operationally)

In operational terms, a "robot" is one deployed Gopherbot instance bound to one custom configuration repository and its runtime environment.

- The robot process starts from local environment variables (for example via `.env` or orchestrator-provided env).
- `GOPHER_CUSTOM_REPOSITORY` identifies the robot's config repository; this repository defines the robot's behavior (`conf/robot.yaml`, plugin/job/task config, etc.).
- When startup detects a configured repository but local config is missing, it enters bootstrap mode and clones that repository, then restarts as a configured robot (see `aidocs/STARTUP_FLOW.md` and `gojobs/go-bootstrap/go_bootstrap_job.go`).
- Secrets in the robot repository can be encrypted and resolved during config expansion with `decrypt`, using the active encryption key initialization flow.
- Connector-specific config like `conf/slack.yaml` is typically part of the custom robot repository, not an installed default under `gopherbot/conf/`.

This means "Clu" (or any named robot) is not just "a connector account"; it is the full config-repo-backed automation unit plus its env/bootstrap lifecycle.

# AI Docs (Gopherbot)

Start here to orient yourself in the repo; read aidocs/COMPONENT_MAP.md first.

**Navigation note**: Documentation references use function and type names (e.g., `bot/handler.go` func `IncomingMessage`) rather than line numbers. Use grep or your editor's symbol search to locate referenced code.

## Table of contents

- `aidocs/COMPONENT_MAP.md` - top-level directory map with entrypoints and representative files.
- `aidocs/EXTENSION_SURFACES.md` - extension types and registration/discovery touchpoints.
- `aidocs/EXTENSION_API.md` - extension language method surface (Robot API).
- `aidocs/JS_METHOD_CHECKLIST.md` - JavaScript extension parity checklist.
- `aidocs/LUA_METHOD_CHECKLIST.md` - Lua extension parity checklist.
- `aidocs/JS_HTTP_API.md` - JavaScript HTTP API design notes.
- `aidocs/LUA_HTTP_API.md` - Lua HTTP API design notes.
- `aidocs/SLACK_CONNECTOR.md` - Slack connector dependency/API notes.
- `aidocs/SSH_CONNECTOR.md` - SSH connector behavior and protocol notes.
- `aidocs/DEV_CONTAINER.md` - dev container build + editor tooling notes.
- `aidocs/TESTING_CURRENT.md` - current integration test harness and test case structure.
- `aidocs/PIPELINE_LIFECYCLE.md` - incoming message to pipeline start flow.
- `aidocs/SCHEDULER_FLOW.md` - cron scheduler to pipeline start flow.
- `aidocs/EXECUTION_SECURITY_MODEL.md` - task execution/threading + privilege separation model.
- `aidocs/multiprocess/...` - slice-by-slice design/impact artifacts for the multiprocess execution epic.

## Routing guide

- Core entrypoint: `main.go` (func `main`) calls `bot.ProcessRegistrations()` and `bot.Start()`; see `bot/start.go` (func `Start`).
- Startup flow details: `aidocs/STARTUP_FLOW.md` (entrypoints listed as `main.go` → `bot.Start()` → `initBot()` → `run()` with `initBot`/`run` in `bot/bot_process.go`).
- Default core engine configuration: `conf/README.md` and `conf/robot.yaml`.
- Initial configuration templates for user robots: `robot.skel/README.md`, `robot.skel/conf/robot.yaml`
- Go compiled-in extension registrations: `robot/registrations.go` (funcs `RegisterPlugin`, `RegisterJob`, `RegisterTask`).
- Script plugin examples: `plugins/README.txt`, `plugins/welcome.lua`, `plugins/samples/`.
- Extension surface map: `aidocs/EXTENSION_SURFACES.md`.
- Extension language method surface: `aidocs/EXTENSION_API.md`.
- JavaScript method checklist: `aidocs/JS_METHOD_CHECKLIST.md`.
- JavaScript HTTP API notes: `aidocs/JS_HTTP_API.md`.
- Dev container and editor tooling: `aidocs/DEV_CONTAINER.md`.
- Testing harness notes: `aidocs/TESTING_CURRENT.md`.
- Incoming message pipeline flow: `aidocs/PIPELINE_LIFECYCLE.md`.
- Scheduled job pipeline flow: `aidocs/SCHEDULER_FLOW.md`.
- Execution/threading/privsep model: `aidocs/EXECUTION_SECURITY_MODEL.md`.
