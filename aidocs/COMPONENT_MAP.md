# Component Map (Top-Level Directories)

Entries cite files like `main.go` and symbols like `Start` in `bot/start.go` for quick verification.

## aidocs/

- AI-focused docbase root: `aidocs/README.md`
- Top-level component map: `aidocs/COMPONENT_MAP.md`
- Startup flow narrative: `aidocs/STARTUP_FLOW.md`
- High-level v3 goals (project-level): `GOALS_v3.md`
- v3 compatibility priorities and migration policy: `aidocs/V3_COMPATIBILITY_CONTRACT.md`
- Execution/threading/security model: `aidocs/EXECUTION_SECURITY_MODEL.md`
- Interpreter notes: `aidocs/INTERPRETERS.md`.
- Extension surface map: `aidocs/EXTENSION_SURFACES.md`.
- Test harness overview: `aidocs/TESTING_CURRENT.md`.
- Incoming message pipeline flow: `aidocs/PIPELINE_LIFECYCLE.md`.
- Scheduled job pipeline flow: `aidocs/SCHEDULER_FLOW.md`.
- AI-maintained backlog: `aidocs/TODO.md`.
- Active workstream indexes: `aidocs/multi-protocol/README.md`, `aidocs/multiprocess/README.md`.
- Archived historical slice artifacts: `aidocs/archive/` (reference only).

## bot/

- Engine entrypoints: `bot/start.go` (func `Start`), `bot/bot_process.go` (funcs `initBot`, `run`, `stop`).
- Runtime connector orchestration: `bot/connector_runtime.go` (runtime manager, protocol routing, lifecycle controls).
- Bot-side connector capability/registration consumption: `bot/connector_capabilities.go` (shared registration lookup + test overrides).
- Bot-side provider registration consumption: `bot/provider_registrations.go` (shared brain/history registration lookup + test overrides).
- Pipeline execution + privilege separation internals: `bot/run_pipelines.go`, `bot/task_execution.go`, `bot/task_execution_child.go`, `bot/pipeline_rpc.go`, `bot/pipeline_rpc_interpreter.go`, `bot/pipeline_rpc_javascript.go`, `bot/pipeline_rpc_yaegi.go`, `bot/calltask.go`, `bot/privsep.go`.
- Startup mode and config loading: `bot/config_load.go` (funcs `detectStartupMode`, `getConfigFile`), `bot/conf.go` (func `loadConfig`).
- Runtime git branch observability: `bot/git_runtime.go` (startup capture + runtime snapshot for info/admin commands), with privileged sync task registration in `bot/pipe_tasks.go` (`git-sync-state`).
- AI-dev endpoint/auth helpers: `bot/aidev.go` (token + `.aiport`) and `bot/aidev_http.go` (authenticated `send_message` / `get_messages` routing).
- Internal module initialization: `bot/modules_init.go` (func `initializeModules`) — initializes ssh-agent, ssh-git-helper, and yaegi interpreter modules.
- Built-in connectors (not under `connectors/`): `bot/term_connector.go` (registers `"terminal"` through `robot.RegisterConnector`), `bot/null_connector.go` (registers `"nullconn"` through `robot.RegisterConnector`).

## brains/

- SimpleBrain providers are registered via `robot.RegisterSimpleBrain` in `brains/dynamodb/static.go` and `brains/cloudflarekv/static.go`.
- Provider implementations: `brains/dynamodb/dynamobrain.go` (func `provider`, methods `Store`, `Retrieve`) and `brains/cloudflarekv/cloudflarekvbrain.go` (func `provider`).

## cmd/

- MCP lifecycle helper binary: `cmd/gopherbot-mcp/main.go` (stdio MCP server with robot lifecycle/status/inventory/readiness/log tools (`start_robot`, `stop_robot`, `restart_robot`, `robot_status`, `wait_robot_ready`, `list_robots`, `cleanup_stale_state`, `tail_robot_log`, `read_robot_log`) plus AI-dev interaction tools (`send_message`, `get_messages`, `send_as_robot`)).

## conf/

- Default configuration: `conf/README.md`, `conf/robot.yaml`, `conf/protocols/terminal.yaml`.
- Brain provider defaults: `conf/brains/*.yaml` (`BrainConfig`).
- History provider defaults: `conf/history/*.yaml` (`HistoryConfig`).
- Default job/plugin config examples live under `conf/jobs/` and `conf/plugins/` (e.g., `conf/jobs/pause-notifies.yaml`, `conf/plugins/builtin-help.yaml`).

## connectors/

- Slack connector registration + init: `connectors/slack/static.go` (calls `robot.RegisterConnector("slack", Initialize, robot.ConnectorCapabilities{HiddenCommands: true})`), `connectors/slack/connect.go` (func `Initialize`; connector-local `ProtocolConfig.UserMap` identity mapping).
- Test connector registration + runtime: `connectors/test/init.go` (calls `robot.RegisterConnector("test", Initialize, robot.ConnectorCapabilities{HiddenCommands: true})`; connector-local `ProtocolConfig.Users` identity mapping), `connectors/test/connector.go` (method `(*TestConnector).Run`).
- SSH connector registration + runtime: `connectors/ssh/static.go` (calls `robot.RegisterConnector("ssh", Initialize, robot.ConnectorCapabilities{HiddenCommands: true})`), `connectors/ssh/connector.go` (method `(*sshConnector).Run`; connector-local `ProtocolConfig.UserKeys` list identity mapping).

## gojobs/

- Compiled Go jobs: `gojobs/go-bootstrap/go_bootstrap_job.go` (init calls `robot.RegisterJob`, handler `bootstrapHandler`).

## goplugins/

- Compiled Go plugins: `goplugins/help/help.go` (init calls `robot.RegisterPlugin`, handler `help`).

## gotasks/

- Compiled Go tasks: `gotasks/ssh-agent/ssh_agent_task.go` (init calls `robot.RegisterTask`, handler `sshAgentTask`).

## helpers/

- Utility scripts: `helpers/vault-password.sh`, `helpers/deprecated/ssh-askpass.sh`.

## history/

- History provider registration: `history/file/static.go` (calls `robot.RegisterHistoryProvider("file", provider)`).
- File-backed implementation: `history/file/filehistory.go` (methods `NewLog`, `GetLog`, `GetLogURL`).

## jobs/

- Shell job scripts: `jobs/logrotate.sh`.
- Go job entrypoint example: `jobs/updatecfg/updatecfg.go` (func `JobHandler`).
- AI conversation retention prune job: `jobs/go-openai-prune/go_openai_prune_job.go` (func `JobHandler`).
- SSH demo welcome trigger job: `jobs/go-welcome-join/welcome_join.go` (func `JobHandler`).

## lib/

- Plugin language libraries: `lib/README.txt`, `lib/gopherbot_v1.sh`, `lib/gopherbot_v1.py`, `lib/gopherbot_v1.rb`, `lib/gopherbot_v1.js`, `lib/gopherbot_v1.lua`, `lib/GopherbotV1.jl`.

## licenses/

- License texts: `licenses/README.txt`, `licenses/Go-LICENSE`, `licenses/aescrypt.txt`.

## modules/

- Internal modules with init hooks: `modules/ssh-agent/ssh_agent_module.go` (func `Initialize`), `modules/ssh-git-helper/ssh_git_helper_module.go` (func `Initialize`), `modules/yaegi-dynamic-go/yaegi_init.go` (func `Initialize`).

## plugins/

- External script plugins and samples: `plugins/README.txt`, `plugins/welcome.lua`, `plugins/welcome.sh`, `plugins/samples/README.txt`.
- OpenAI fallback plugin: `plugins/go-openai-fallback/ai.go` (func `PluginHandler`).

## resources/

- Deployment and service artifacts: `resources/deploy-gopherbot.yaml`, `resources/robot.service`, `resources/user-robot.service`.
- Container build assets: `resources/containers/build-dev.sh`, `resources/containers/build-base.sh`, `resources/containers/build-min.sh`.
- Dev container specs + IDE assets: `resources/containers/containerfile.base`, `resources/containers/containerfile.dev`, `resources/containers/assets/jsconfig.json`, `resources/containers/assets/gopherbot.code-workspace`.

## robot/

- Shared modular contract surface: `robot/README.md`.
- Go extension registrations: `robot/registrations.go` (funcs `RegisterPlugin`, `RegisterJob`, `RegisterTask`).
- Connector registrations + capabilities: `robot/connectors.go` (`RegisterConnector`, `ConnectorCapabilities`, `HiddenCommandFormatter`).
- Brain-provider registrations: `robot/brains.go` (`RegisterSimpleBrain`).
- History-provider registrations: `robot/history_providers.go` (`RegisterHistoryProvider`).
- Connector contracts and connector-side APIs: `robot/connector_defs.go` (`Connector`, `ConnectorAPIProvider`, `Injector`, `MessageSource`).
- Shared wrapping utility used by engine and connectors: `robot/wrap.go` (`Wrapper`, `NewWrapper`, `Wrap`).

## robot.skel/

- Default robot skeleton: `robot.skel/README.md`, `robot.skel/conf/robot.yaml`, `robot.skel/conf/environments/development.yaml`, `robot.skel/conf/protocols/ssh.yaml`.

## tasks/

- External task scripts: `tasks/exec.sh`, `tasks/reply.sh`, `tasks/notify.sh`.

## test/

- Integration test docs/configs: `test/README.md`, `test/bot_integration_test.go`, `test/jsfull/conf/robot.yaml`.
