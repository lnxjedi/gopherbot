# Extension Surfaces

Concise map of extension types, where they live, and how they register/discover. Every claim points to a concrete file or symbol; unknowns are marked TODO.

## Connectors

- Where: protocol connectors under `connectors/`, plus built-ins like `bot/term_connector.go` and `bot/null_connector.go`.
- Registration: `bot/bot_process.go` (func `RegisterConnector`) called from connector init (e.g., `connectors/slack/static.go` calls `bot.RegisterConnector("slack", Initialize)`).
- Selection: `bot/conf.go` (type `ConfigLoader` field `Protocol`) reads `conf/robot.yaml`.
- Examples: `connectors/slack/connect.go` (func `Initialize`), `connectors/test/init.go` (func `Initialize`), `bot/term_connector.go` (calls `RegisterConnector("terminal", Initialize)`).

## Brains (SimpleBrain providers)

- Where: built-ins in `bot/membrain.go` and `bot/filebrain.go`, plus providers under `brains/`.
- Registration: `bot/brain.go` (func `RegisterSimpleBrain`) called from provider `init()` (e.g., `brains/dynamodb/static.go`, `brains/cloudflarekv/static.go`, `bot/membrain.go`).
- Selection: `bot/conf.go` (type `ConfigLoader` field `Brain`) reads `conf/robot.yaml`.
- Examples: `brains/dynamodb/dynamobrain.go` (func `provider`), `brains/cloudflarekv/cloudflarekvbrain.go` (func `provider`), `bot/filebrain.go` (func `fbprovider`).

## Script plugins (external executables)

- Where: scripts live under `plugins/` (e.g., `plugins/welcome.sh`, `plugins/weather.rb`, `plugins/chuck.rb`).
- Discovery: `conf/robot.yaml` defines `ExternalPlugins`, loaded into `bot/conf.go` (type `ConfigLoader` field `ExternalPlugins`), then turned into tasks in `bot/taskconf.go` (func `addExternalTask`).
- Execution: external plugins run via `exec.Command` in `bot/calltask.go` (external branch after `.go/.lua/.js` checks).
- Examples: `plugins/welcome.sh`, `plugins/weather.rb`, `plugins/samples/hello.sh`.

## External jobs (scripted)

- Where: job scripts live under `jobs/` (e.g., `jobs/backup.sh`, `jobs/restore.sh`).
- Discovery: `conf/robot.yaml` defines `ExternalJobs`, loaded into `bot/conf.go` (type `ConfigLoader` field `ExternalJobs`), then turned into tasks in `bot/taskconf.go` (func `addExternalTask`).
- Execution: external jobs run via `exec.Command` in `bot/calltask.go` (external branch after `.go/.lua/.js` checks).
- Examples: `jobs/backup.sh`, `jobs/restore.sh`.

## External tasks (scripted)

- Where: task scripts live under `tasks/` (e.g., `tasks/exec.sh`, `tasks/notify.sh`).
- Discovery: `conf/robot.yaml` defines `ExternalTasks`, loaded into `bot/conf.go` (type `ConfigLoader` field `ExternalTasks`), then turned into tasks in `bot/taskconf.go` (func `addExternalTask`).
- Execution: external tasks run via `exec.Command` in `bot/calltask.go` (external branch after `.go/.lua/.js` checks).
- Examples: `tasks/exec.sh`, `tasks/notify.sh`.

## Go plugins

- Where: Compiled-in Go plugin implementations live under `goplugins/`.
- Registration: `robot/registrations.go` (func `RegisterPlugin`) called in plugin `init()` (e.g., `goplugins/help/help.go`), collected and wired by `bot/registrations.go` (func `ProcessRegistrations`), which is invoked in `main.go` (func `main`).
- Examples: `goplugins/help/help.go` (func `init`), `goplugins/duo/duo.go` (func `init`), `goplugins/groups/groups.go` (func `init`).

## Go tasks

- Where: Go task implementations live under `gotasks/`.
- Registration: `robot/registrations.go` (func `RegisterTask`) called in task `init()` (e.g., `gotasks/ssh-agent/ssh_agent_task.go`), collected by `bot/registrations.go` (func `ProcessRegistrations`) and initialized from `main.go` (func `main`).
- Examples: `gotasks/ssh-agent/ssh_agent_task.go` (func `init`), `gotasks/git-command/git_command_task.go` (func `init`), `gotasks/ssh-git-helper/ssh_git_helper_task.go` (func `init`).

## Go jobs

- Where: Go job implementations live under `gojobs/`.
- Registration: `robot/registrations.go` (func `RegisterJob`) called in job `init()` (e.g., `gojobs/go-bootstrap/go_bootstrap_job.go`), collected by `bot/registrations.go` (func `ProcessRegistrations`) and initialized from `main.go` (func `main`).
- Examples: `gojobs/go-bootstrap/go_bootstrap_job.go` (func `init`).

## Built-in interpreter-based extensions (Lua/JS/Go)

- Where: interpreter modules live under `modules/lua/`, `modules/javascript/`, and `modules/yaegi-dynamic-go/`; example script sources live under `plugins/` (e.g., `plugins/samples/hello.lua`, `plugins/samples/hello.js`, `plugins/go-lists/lists.go`).
- Dispatch: `bot/calltask.go` selects interpreter by file extension (`.lua`, `.js`, `.go`) and calls `modules/lua/call_extension.go` (func `CallExtension`), `modules/javascript/call_extension.go` (func `CallExtension`), or `modules/yaegi-dynamic-go/yaegi_dynamic.go` (funcs `RunPluginHandler`, `RunJobHandler`, `RunTaskHandler`).
- Examples: Lua `plugins/samples/hello.lua`, JavaScript `plugins/samples/hello.js`, dynamic Go `plugins/go-lists/lists.go` (funcs `Configure`, `PluginHandler`).
- See also: `aidocs/INTERPRETERS.md`.

## Build Mechanics Note

The file `modules.go` carries a `//go:build test` constraint, but production builds still include it because the Makefile explicitly names it on the build command line (`go build ... main.go modules.go`), which overrides build tags. This pattern exists so that `go test ./...` doesn't double-import extensions that test files already import directly.
