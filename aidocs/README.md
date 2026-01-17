# AI Docs (Gopherbot)

Start here to orient yourself in the repo; read aidocs/COMPONENT_MAP.md first.

## Table of contents

- `aidocs/COMPONENT_MAP.md` - top-level directory map with entrypoints and representative files.
- `aidocs/EXTENSION_SURFACES.md` - extension types and registration/discovery touchpoints.
- `aidocs/TESTING_CURRENT.md` - current integration test harness and test case structure.
- `aidocs/PIPELINE_LIFECYCLE.md` - incoming message to pipeline start flow.

## Routing guide

- Core entrypoint: `main.go` (func `main`) calls `bot.ProcessRegistrations()` and `bot.Start()`; see `bot/start.go` (func `Start`).
- Startup flow details: `aidocs/STARTUP_FLOW.md` (entrypoints listed as `main.go` → `bot.Start()` → `initBot()` → `run()` with `initBot`/`run` in `bot/bot_process.go`).
- Default core engine configuration: `conf/README.md` and `conf/robot.yaml`.
- Initial configuration templates for user robots: `robot.skel/README.md`, `robot.skel/conf/robot.yaml`
- Go compiled-in extension registrations: `robot/registrations.go` (funcs `RegisterPlugin`, `RegisterJob`, `RegisterTask`).
- Script plugin examples: `plugins/README.txt`, `plugins/welcome.lua`, `plugins/samples/`.
- Extension surface map: `aidocs/EXTENSION_SURFACES.md`.
- Testing harness notes: `aidocs/TESTING_CURRENT.md`.
- Incoming message pipeline flow: `aidocs/PIPELINE_LIFECYCLE.md`.
