# Extension Surfaces

Concise map of extension types, where they live, and how they register/discover. Every claim points to a concrete file or symbol; unknowns are marked TODO.

## Shared Boundary

- `robot/` is the shared contract surface for modular components that plug into the engine.
- Connectors, brains, and history providers should depend on `robot/`, not `bot/`.
- `bot/` consumes those registrations and contracts; it should not be the package external modular components need to import.

## Connectors

- Where: protocol connectors under `connectors/`, plus built-ins like `bot/term_connector.go` and `bot/null_connector.go`.
- Registration: `robot/connectors.go` (func `RegisterConnector`) called from connector init (for example `connectors/slack/static.go` calls `robot.RegisterConnector("slack", Initialize)`).
- Initialization: connector `Initialize(...)` returns `robot.InitializedConnector{Connector, Capabilities}`, letting connectors compute runtime capabilities after reading protocol config.
- Connector init also receives shared robot identity through `robot.Handler.GetBotInfo()`, so local connectors can derive bot-addressed behavior from `BotInfo` without duplicating bot-name fields in `ProtocolConfig`.
- Capabilities: `robot/connectors.go` (types `InitializedConnector`, `ConnectorCapabilities`) holds engine-owned connector capability flags such as `HiddenCommands`.
- Optional connector-owned hidden-help rendering hook: `robot/connectors.go` (interface `HiddenCommandFormatter`), consumed in `bot/connector_capabilities.go` so engine help/fallback and hidden-command denials can render a concrete protocol-correct hidden command such as `/clu help ping`.
- Selection: `bot/conf.go` (type `ConfigLoader` fields `PrimaryProtocol`/`DefaultProtocol`) reads `conf/robot.yaml`; connector-specific `ProtocolConfig` is loaded from `conf/protocols/<protocol>.yaml`.
- Examples: `connectors/slack/connect.go` (func `Initialize`), `connectors/test/init.go` (func `Initialize`), `bot/term_connector.go` (registers `"terminal"` and returns hidden-command capability from `Initialize(...)`).

## Brains (SimpleBrain providers)

- Where: built-ins in `bot/membrain.go` and `bot/filebrain.go`, plus providers under `brains/`.
- Registration: `robot/brains.go` (func `RegisterSimpleBrain`) called from provider `init()` (e.g., `brains/dynamodb/static.go`, `brains/cloudflarekv/static.go`, `bot/membrain.go`).
- Selection: `bot/conf.go` (type `ConfigLoader` field `Brain`) reads `conf/robot.yaml`.
- Examples: `brains/dynamodb/dynamobrain.go` (func `provider`), `brains/cloudflarekv/cloudflarekvbrain.go` (func `provider`), `bot/filebrain.go` (func `fbprovider`).

## History Providers

- Where: built-ins in `bot/memhistory.go`, plus providers under `history/`.
- Registration: `robot/history_providers.go` (func `RegisterHistoryProvider`) called from provider `init()` (for example `history/file/static.go`, `bot/memhistory.go`).
- Selection: `bot/conf.go` (type `ConfigLoader` field `HistoryProvider`) reads `conf/robot.yaml`.
- Examples: `history/file/filehistory.go` (func `provider`), `bot/memhistory.go` (func `mhprovider`).

## Script plugins (external executables)

- Where: scripts live under `plugins/` (e.g., `plugins/welcome.sh`, `plugins/weather.rb`, `plugins/chuck.rb`).
- Discovery: `conf/robot.yaml` defines `ExternalPlugins`, loaded into `bot/conf.go` (type `ConfigLoader` field `ExternalPlugins`), then turned into tasks in `bot/taskconf.go` (func `addExternalTask`).
- Execution: external plugins run via `exec.Command` in `bot/calltask.go` (external branch after `.go/.lua/.js` checks).
- Examples: `plugins/welcome.sh`, `plugins/weather.rb`, `plugins/samples/hello.sh`.

## External jobs (scripted)

- Where: job scripts live under `jobs/` (e.g., `jobs/logrotate.sh`).
- Discovery: `conf/robot.yaml` defines `ExternalJobs`, loaded into `bot/conf.go` (type `ConfigLoader` field `ExternalJobs`), then turned into tasks in `bot/taskconf.go` (func `addExternalTask`).
- Execution: external jobs run via `exec.Command` in `bot/calltask.go` (external branch after `.go/.lua/.js` checks).
- Examples: `jobs/logrotate.sh`.

## External tasks (scripted)

- Where: task scripts live under `tasks/` when they still rely on an external executable entrypoint (e.g., `tasks/remote-exec.sh`, `tasks/setworkdir.sh`).
- Discovery: `conf/robot.yaml` defines `ExternalTasks`, loaded into `bot/conf.go` (type `ConfigLoader` field `ExternalTasks`), then turned into tasks in `bot/taskconf.go` (func `addExternalTask`).
- Execution: external tasks run via `exec.Command` in `bot/calltask.go` (external branch after `.go/.lua/.js` checks).
- Examples: `tasks/remote-exec.sh`, `tasks/setworkdir.sh`.

## Go plugins

- Where: Compiled-in Go plugin implementations live under `goplugins/`.
- Registration: `robot/registrations.go` (func `RegisterPlugin`) called in plugin `init()` (e.g., `goplugins/help/help.go`), collected and wired by `bot/registrations.go` (func `ProcessRegistrations`), which is invoked in `main.go` (func `main`).
- Examples: `goplugins/help/help.go` (func `init`), `goplugins/duo/duo.go` (func `init`), `goplugins/groups/groups.go` (func `init`).

Shipped identity onboarding note:
- `plugins/go-github-link/github_link.go` implements `github-link` as an external Go plugin with shipped config in `conf/plugins/github-link.yaml`, but it is intended as an explicit custom-robot opt-in once the owner supplies credentials and enables it.

## Go tasks

- Where: Go task implementations live under `gotasks/`.
- Registration: `robot/registrations.go` (func `RegisterTask`) called in task `init()` (e.g., `gotasks/ssh-agent/ssh_agent_task.go`), collected by `bot/registrations.go` (func `ProcessRegistrations`) and initialized from `main.go` (func `main`).
- Examples: `gotasks/ssh-agent/ssh_agent_task.go` (func `init`), `gotasks/git-command/git_command_task.go` (func `init`), `gotasks/ssh-git-helper/ssh_git_helper_task.go` (func `init`).

## Go jobs

- Where: Go job implementations live under `gojobs/`.
- Registration: `robot/registrations.go` (func `RegisterJob`) called in job `init()` (e.g., `gojobs/go-bootstrap/go_bootstrap_job.go`), collected by `bot/registrations.go` (func `ProcessRegistrations`) and initialized from `main.go` (func `main`).
- Examples: `gojobs/go-bootstrap/go_bootstrap_job.go` (func `init`).

## Built-in interpreter-based extensions (Lua/JS/Gsh/Go)

- Where: interpreter modules live under `modules/lua/`, `modules/javascript/`, `modules/gsh/`, and `modules/yaegi-dynamic-go/`; example script sources live under `plugins/` (e.g., `plugins/samples/hello.lua`, `plugins/samples/hello.js`, `plugins/samples/hello.gsh`, `plugins/go-lists/lists.go`).
- Dispatch: `bot/calltask.go` selects interpreter by file extension (`.lua`, `.js`, `.gsh`, `.go`) and routes through `modules/lua/call_extension.go` (func `CallExtension`), `modules/javascript/call_extension.go` (func `CallExtension`), `modules/gsh/call_extension.go` (func `CallExtension`), or `modules/yaegi-dynamic-go/yaegi_dynamic.go` (funcs `RunPluginHandler`, `RunJobHandler`, `RunTaskHandler`).
- Examples: Lua `plugins/samples/hello.lua`, JavaScript `plugins/samples/hello.js`, Gopherbot shell `plugins/samples/hello.gsh`, shipped Gopherbot shell defaults like `plugins/admin.gsh` and `tasks/status.gsh`, dynamic Go `plugins/go-lists/lists.go` (funcs `Configure`, `PluginHandler`).
- See also: `aidocs/INTERPRETERS.md`.

Shared identity method surface:
- Interpreter-backed extensions can call `GetIdentityCredential`, `LinkOAuth2Identity`, and `UnlinkIdentity` through the same robot API surface exposed by `modules/javascript/`, `modules/lua/`, `modules/gsh/`, and `bot/pipeline_rpc_interpreter.go`.
- Provider-backed identity access is still explicitly scoped: the calling task/plugin/job must have the provider's credential `ParameterSet` attached, or the engine returns `IdentityConfigError` and logs the missing attachment.

## Build Mechanics Note

The file `modules.go` carries a `//go:build test` constraint, but production builds still include it because the Makefile explicitly names it on the build command line (`go build ... main.go modules.go`), which overrides build tags. This pattern exists so that `go test ./...` doesn't double-import extensions that test files already import directly.
