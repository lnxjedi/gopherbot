# Component Map (Top-Level Directories)

Entries cite files like `main.go` and symbols like `Start` in `bot/start.go` for quick verification.

## aidocs/

- AI-focused docbase root: `aidocs/README.md`
- Top-level component map: `aidocs/COMPONENT_MAP.md`
- Startup flow narrative: `aidocs/STARTUP_FLOW.md`
- High-level v3 goals (project-level): `GOALS_v3.md`
- v3 compatibility priorities and migration policy: `aidocs/V3_COMPATIBILITY_CONTRACT.md`
- Execution/threading/security model: `aidocs/EXECUTION_SECURITY_MODEL.md`
- macOS privilege-separation one-shot child process plan: `aidocs/macos-privsep.md`.
- Setup UX/style conventions for guided onboarding flows: `aidocs/setup-style-guide.md`
- Interpreter notes: `aidocs/INTERPRETERS.md`.
- Extension surface map: `aidocs/EXTENSION_SURFACES.md`.
- Environment-scoped secrets/variables behavior and design:
  `aidocs/SECRETS_VARIABLES_ENVIRONMENT_DESIGN.md`.
- Test harness overview: `aidocs/TESTING_CURRENT.md`.
- Process-backed integration harness migration plan:
  `aidocs/INTEGRATION_HARNESS_PLAN.md`.
- OAuth2 refresh registry, brain schema, and token lifecycle: `aidocs/OAUTH2_TOKEN_MANAGEMENT.md`.
- Incoming message pipeline flow: `aidocs/PIPELINE_LIFECYCLE.md`.
- SimpleMatcher diagnostic routing design: `aidocs/SIMPLE_MATCHER_DIAGNOSTICS.md`.
- Scheduled job pipeline flow: `aidocs/SCHEDULER_FLOW.md`.
- AI-maintained backlog: `aidocs/TODO.md`.
- Active workstream indexes: `aidocs/multi-protocol/README.md`, `aidocs/multiprocess/README.md`.
- Archived historical slice artifacts: `aidocs/archive/` (reference only).

## bot/

- Engine entrypoints: `bot/start.go` (func `Start`), `bot/bot_process.go` (funcs `initBot`, `run`, `stop`), `bot/startup_ready.go` (startup readiness signal for integration harnesses).
- Runtime connector orchestration: `bot/connector_runtime.go` (runtime manager, protocol routing, lifecycle controls).
- Bot-side connector capability/registration consumption: `bot/connector_capabilities.go` (shared registration lookup, runtime capability lookup, and test overrides).
- Connector/brain/history handler implementation: `bot/handler.go` (implements shared `robot.Handler`, including `GetBotInfo()` for connector init).
- Bot-side provider registration consumption: `bot/provider_registrations.go` (shared brain/history registration lookup + test overrides).
- Pipeline execution + privilege separation internals: `bot/run_pipelines.go`, `bot/task_execution.go`, `bot/task_execution_child.go`, `bot/pipeline_rpc.go`, `bot/pipeline_rpc_interpreter.go`, `bot/pipeline_rpc_javascript.go`, `bot/pipeline_rpc_gsh.go`, `bot/pipeline_rpc_yaegi.go`, `bot/calltask.go`, `bot/privsep.go`, `bot/privsep_darwin.go`, `bot/privsep_process.go`.
- Startup mode and config loading: `bot/config_load.go` (funcs `detectStartupMode`, `getConfigFile`), `bot/conf.go` (func `loadConfig`).
- Runtime git branch observability: `bot/git_runtime.go` (startup capture + runtime snapshot for info/admin commands), with privileged sync task registration in `bot/pipe_tasks.go` (`git-sync-state`).
- AI-dev endpoint/auth helpers: `bot/aidev.go` (token + `.aiport`) and `bot/aidev_http.go` (authenticated `send_message` / `get_messages` routing).
- Internal module initialization: `bot/modules_init.go` (func `initializeModules`) — initializes ssh-agent, ssh-git-helper, and Yaegi runtime support including the shared `$GOPHER_HOME/.yaegi-gopath` tree used by interpreted Go extensions.
- OAuth2 token manager + internal refresh-registry handling: `bot/oauth2.go`, `bot/oauth2_types.go`.
- Built-in connectors (not under `connectors/`): `bot/term_connector.go` (registers `"terminal"` through `robot.RegisterConnector`), `bot/null_connector.go` (registers `"nullconn"` through `robot.RegisterConnector`).

## brains/

- SimpleBrain providers are registered via `robot.RegisterSimpleBrain` in `brains/dynamodb/static.go`, `brains/cloudflarekv/static.go`, and `brains/firestore/static.go`.
- Provider implementations: `brains/dynamodb/dynamobrain.go` (func `provider`, methods `Store`, `Retrieve`), `brains/cloudflarekv/cloudflarekvbrain.go` (func `provider`), and `brains/firestore/firestorebrain.go` (func `provider`, methods `Store`, `Retrieve`, `List`, `Delete`).

## cmd/

- MCP lifecycle helper binary: `cmd/gopherbot-mcp/main.go` (stdio MCP server with robot lifecycle/status/inventory/readiness/log tools (`start_robot`, `stop_robot`, `restart_robot`, `robot_status`, `wait_robot_ready`, `list_robots`, `cleanup_stale_state`, `tail_robot_log`, `read_robot_log`), AI-dev interaction tools (`send_message`, `get_messages`, `send_as_robot`), and integration runner tools (`list_integration_suites`, `run_integration_suite`, `read_integration_result`)).
- Process-backed integration suite runner:
  `cmd/gopherbot-integration/main.go` (CLI, artifact setup, scripted connector
  driver, YAML-loaded suite execution, suite metadata selectors).

## conf/

- Default configuration: `conf/README.md`, `conf/robot.yaml`, `conf/protocols/terminal.yaml`.
- Shipped OAuth2/GitHub linker command config: `conf/plugins/github-link.yaml`.
- Installed connector defaults plus inert setup templates: `conf/protocols/googlechat.yaml`, `conf/protocols/slack.yaml.sample`, `conf/protocols/ssh.yaml`, `conf/protocols/terminal.yaml`, `conf/protocols/nullconn.yaml`. Active robot-specific changes belong under `custom/conf/`.
- Brain provider defaults: `conf/brains/*.yaml` (`BrainConfig`).
- History provider defaults: `conf/history/*.yaml` (`HistoryConfig`).
- Default job/plugin config examples live under `conf/jobs/` and `conf/plugins/` (e.g., `conf/jobs/pause-notifies.yaml`, `conf/plugins/builtin-help.yaml`).

## connectors/

- Slack connector registration + init: `connectors/slack/static.go` (calls `robot.RegisterConnector("slack", Initialize)`), `connectors/slack/connect.go` (func `Initialize`; connector-local `ProtocolConfig.UserMap` identity mapping, reloads that map via `Reload`, plus slash-command-driven runtime hidden-command capability).
- Google Chat connector registration + init: `connectors/googlechat/static.go` (calls `robot.RegisterConnector("googlechat", Initialize)`), `connectors/googlechat/connect.go` (func `Initialize`; connector-local `ProtocolConfig.UserMap` identity mapping reloadable via `Reload`, shared encrypted Google credential loading, Pub/Sub subscription receive loop, slash-command hidden-command capability, thread-default send behavior, and ambient Workspace Events setup when enabled), with ambient subscription lifecycle + CloudEvent handling in `connectors/googlechat/ambient.go` and `connectors/googlechat/workspaceevents.go`.
- Test connector registration + runtime: `connectors/test/init.go` (calls `robot.RegisterConnector("test", Initialize)`; connector-local `ProtocolConfig.Users` identity mapping), `connectors/test/connector.go` (method `(*TestConnector).Run`).
- SSH connector registration + runtime: `connectors/ssh/static.go` (calls `robot.RegisterConnector("ssh", Initialize)`), `connectors/ssh/connector.go` (methods `(*sshConnector).Run` and `(*sshConnector).Reload`; connector-local `ProtocolConfig.UserKeys` list identity mapping plus runtime hidden-command capability).

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

## internal/

- Internal shared cloud helpers: `internal/gcloud/credentials.go` (service-account validation, encrypted credential loading callback wiring, and Google client options for engine-owned components).

## jobs/

- Built-in shell/runtime job scripts: `jobs/install-libs.gsh`.
- External shell job scripts: `jobs/logrotate.sh`.
- Go job entrypoint example: `jobs/updatecfg/updatecfg.go` (func `JobHandler`).
- AI conversation retention prune job: `jobs/go-openai-prune/go_openai_prune_job.go` (func `JobHandler`).
- SSH onboarding welcome trigger job: `jobs/go-welcome-join/welcome_join.go` (func `JobHandler`).
- SSH onboarding resume-on-join job: `jobs/go-resume-setup/resume_setup.go` (func `JobHandler`).

## lib/

- Plugin language libraries: `lib/README.txt`, `lib/gopherbot_v1.sh`, `lib/gopherbot_v1.py`, `lib/gopherbot_v1.rb`, `lib/gopherbot_v1.js`, `lib/gopherbot_v1.lua`, `lib/GopherbotV1.jl`.
- Shared Go helper module root: `lib/go.mod` declares `module gopherbot.internal/lib` for interpreted-Go imports from installed libraries.
- Shared Go onboarding flow/state helpers: `lib/newrobotflow/onboarding.go`.

## licenses/

- License texts: `licenses/README.txt`, `licenses/Go-LICENSE`, `licenses/aescrypt.txt`, `licenses/mvdan-sh.txt`, `licenses/u-root.txt`, `licenses/gojq.txt`, with summary notices in `LEGAL.md`.

## modules/

- Internal modules with init hooks: `modules/ssh-agent/ssh_agent_module.go` (func `Initialize`), `modules/ssh-git-helper/ssh_git_helper_module.go` (func `Initialize`), `modules/gsh/call_extension.go` (func `CallExtension`), `modules/yaegi-dynamic-go/yaegi_init.go` (func `Initialize`).
- Yaegi shared GOPATH staging + runtime entrypoints: `modules/yaegi-dynamic-go/yaegi_dynamic.go` (`ensureGoPath`, `RunPluginHandler`, `RunJobHandler`, `RunTaskHandler`).

## plugins/

- External and interpreter-backed script plugins/samples: `plugins/README.txt`, `plugins/admin.gsh`, `plugins/ssh-admin.gsh`, `plugins/welcome.lua`, `plugins/welcome.sh`, `plugins/samples/README.txt`, `plugins/samples/hello.gsh`, `plugins/test/shfull.gsh`.
- Shipped OAuth2 onboarding plugin: `plugins/go-github-link/github_link.go`.
- OpenAI fallback plugin: `plugins/go-openai-fallback/ai.go` (func `PluginHandler`).

## resources/

- Deployment and service artifacts: `resources/deploy-gopherbot.yaml`, `resources/robot.service`, `resources/user-robot.service`.
- Container build assets: `resources/containers/build-dev.sh`, `resources/containers/build-base.sh`, `resources/containers/build-min.sh`.
- Dev container specs + IDE assets: `resources/containers/containerfile.base`, `resources/containers/containerfile.dev`, `resources/containers/assets/jsconfig.json`, `resources/containers/assets/gopherbot.code-workspace`.

## robot/

- Shared modular contract surface: `robot/README.md`.
- Go extension registrations: `robot/registrations.go` (funcs `RegisterPlugin`, `RegisterJob`, `RegisterTask`).
- Connector registrations + capabilities: `robot/connectors.go` (`RegisterConnector`, `InitializedConnector`, `ConnectorCapabilities`, `HiddenCommandFormatter`).
- Shared robot identity shape for connector/provider init: `robot/botinfo.go` (`BotInfo`).
- Brain-provider registrations: `robot/brains.go` (`RegisterSimpleBrain`).
- History-provider registrations: `robot/history_providers.go` (`RegisterHistoryProvider`).
- OAuth2 extension API request shape: `robot/oauth2.go`.
- Connector contracts and connector-side APIs: `robot/connector_defs.go` (`Connector` including runtime `Reload`, `ConnectorAPIProvider`, `Injector`, `MessageSource`, and `Handler.ReadEncryptedFile`).
- Shared pure helpers used by engine and connectors: `robot/util/wrap.go` (`Wrapper`, `NewWrapper`, `Wrap`), `robot/util/id.go` (`ExtractID`), `robot/util/basic_markdown_plain.go` (`RenderBasicMarkdownPlain`).

## robot.skel/

- Default robot skeleton: `robot.skel/README.md`, `robot.skel/conf/robot.yaml`, `robot.skel/conf/environments/development.yaml`, `robot.skel/conf/protocols/ssh.yaml`.

## tasks/

- Built-in shell/runtime task scripts: `tasks/exec.gsh`, `tasks/reply.gsh`, `tasks/notify.gsh`, `tasks/status.gsh`.
- External task scripts: `tasks/remote-exec.sh`, `tasks/setworkdir.sh`.

## test/

- Integration test docs/configs: `test/README.md`, `test/bot_integration_test.go`, `test/jsfull/conf/robot.yaml`, `test/sh_full_test.go`, `test/shfull/conf/robot.yaml`.

## integration/

- Process-backed integration harness artifacts and data-driven suite registry:
  `integration/.gitignore`, `integration/suites/` (runner types, YAML loader,
  HTTP fixture), `integration/suites/data/*.yaml` (readable suite definitions).
