Gopherbot is an extensible automation framework designed as a persistent, Go-based chatbot ("robot"). It connects to chat platforms (or the terminal) and executes automation "pipelines" triggered by chat messages, scheduled events, or internal calls. Pipelines consist of compiled Go, interpreted scripts (Lua, JavaScript, dynamic Go), or external executables. A key v3 goal is self-containment, minimizing external tool dependencies. The robot features a pluggable "brain" for persistence and a flexible, template-based configuration system.

# AI Docs (Gopherbot)

Start here to orient yourself in the repo; read aidocs/COMPONENT_MAP.md first.

**Navigation note**: Documentation references use function and type names (e.g., `bot/handler.go` func `IncomingMessage`) rather than line numbers. Use grep or your editor's symbol search to locate referenced code.

## Table of contents

- `aidocs/COMPONENT_MAP.md` - top-level directory map with entrypoints and representative files.
- `aidocs/EXTENSION_SURFACES.md` - extension types and registration/discovery touchpoints.
- `aidocs/EXTENSION_API.md` - extension language method surface (Robot API).
- `aidocs/JS_METHOD_CHECKLIST.md` - JavaScript extension parity checklist.
- `aidocs/TESTING_CURRENT.md` - current integration test harness and test case structure.
- `aidocs/PIPELINE_LIFECYCLE.md` - incoming message to pipeline start flow.
- `aidocs/SCHEDULER_FLOW.md` - cron scheduler to pipeline start flow.

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
- Testing harness notes: `aidocs/TESTING_CURRENT.md`.
- Incoming message pipeline flow: `aidocs/PIPELINE_LIFECYCLE.md`.
- Scheduled job pipeline flow: `aidocs/SCHEDULER_FLOW.md`.
