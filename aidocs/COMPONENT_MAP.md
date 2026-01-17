# Component Map (Top-Level Directories)

Entries cite files like `main.go` and symbols like `Start` in `bot/start.go` for quick verification.

## aidocs/

- AI-focused docbase root: `aidocs/README.md`
- Top-level component map: `aidocs/COMPONENT_MAP.md`
- Startup flow narrative: `aidocs/STARTUP_FLOW.md`
- High-level v3 goals: `aidocs/GOALS_v3.md`
- Interpreter notes: `aidocs/INTERPRETERS.md`.
- Extension surface map: `aidocs/EXTENSION_SURFACES.md`.
- Test harness overview: `aidocs/TESTING_CURRENT.md`.
- Incoming message pipeline flow: `aidocs/PIPELINE_LIFECYCLE.md`.

## bot/

- Engine entrypoints: `bot/start.go` (func `Start`), `bot/bot_process.go` (funcs `initBot`, `run`).
- Startup mode logic: `bot/config_load.go` (func `detectStartupMode` referenced in `aidocs/STARTUP_FLOW.md`).

## brains/

- SimpleBrain providers are registered via `bot.RegisterSimpleBrain` in `brains/dynamodb/static.go` and `brains/cloudflarekv/static.go`.
- Provider implementations: `brains/dynamodb/dynamobrain.go` (func `provider`, methods `Store`, `Retrieve`) and `brains/cloudflarekv/cloudflarekvbrain.go` (func `provider`).

## conf/

- Default configuration: `conf/README.md`, `conf/robot.yaml`, `conf/terminal.yaml`.
- Default job/plugin config examples live under `conf/jobs/` and `conf/plugins/` (e.g., `conf/jobs/backup.yaml`, `conf/plugins/builtin-help.yaml`).

## connectors/

- Slack connector registration + init: `connectors/slack/static.go` (calls `bot.RegisterConnector("slack", Initialize)`), `connectors/slack/connect.go` (func `Initialize`).
- Test connector registration + runtime: `connectors/test/init.go` (calls `bot.RegisterConnector("test", Initialize)`), `connectors/test/connector.go` (method `(*TestConnector).Run`).

## gojobs/

- Compiled Go jobs: `gojobs/go-bootstrap/go_bootstrap_job.go` (init calls `robot.RegisterJob`, handler `bootstrapHandler`).

## goplugins/

- Compiled Go plugins: `goplugins/help/help.go` (init calls `robot.RegisterPlugin`, handler `help`).

## gotasks/

- Compiled Go tasks: `gotasks/ssh-agent/ssh_agent_task.go` (init calls `robot.RegisterTask`, handler `sshAgentTask`).

## helpers/

- Utility scripts: `helpers/vault-password.sh`, `helpers/deprecated/ssh-askpass.sh`.

## history/

- History provider registration: `history/file/static.go` (calls `bot.RegisterHistoryProvider("file", provider)`).
- File-backed implementation: `history/file/filehistory.go` (methods `NewLog`, `GetLog`, `GetLogURL`).

## jobs/

- Shell job scripts: `jobs/backup.sh`, `jobs/restore.sh`.
- Go job entrypoint example: `jobs/updatecfg/updatecfg.go` (func `JobHandler`).

## lib/

- Plugin language libraries: `lib/README.txt`, `lib/gopherbot_v1.sh`, `lib/gopherbot_v1.py`, `lib/gopherbot_v1.rb`, `lib/gopherbot_v1.js`, `lib/gopherbot_v1.lua`, `lib/GopherbotV1.jl`.

## licenses/

- License texts: `licenses/README.txt`, `licenses/Go-LICENSE`, `licenses/aescrypt.txt`.

## modules/

- Internal modules with init hooks: `modules/ssh-agent/ssh_agent_module.go` (func `Initialize`), `modules/ssh-git-helper/ssh_git_helper_module.go` (func `Initialize`), `modules/yaegi-dynamic-go/yaegi_init.go` (func `Initialize`).

## plugins/

- External script plugins and samples: `plugins/README.txt`, `plugins/welcome.lua`, `plugins/welcome.sh`, `plugins/samples/README.txt`.

## resources/

- Deployment and service artifacts: `resources/deploy-gopherbot.yaml`, `resources/robot.service`, `resources/user-robot.service`.
- Container build assets: `resources/containers/build-dev.sh`, `resources/containers/build-base.sh`, `resources/containers/build-min.sh`.

## robot/

- Go extension API: `robot/README.md`, `robot/registrations.go` (funcs `RegisterPlugin`, `RegisterJob`, `RegisterTask`).

## robot.skel/

- Default robot skeleton: `robot.skel/README.md`, `robot.skel/conf/robot.yaml`, `robot.skel/conf/terminal.yaml`, `robot.skel/conf/slack.yaml`, `robot.skel/jobs/hello.sh`.

## tasks/

- External task scripts: `tasks/exec.sh`, `tasks/reply.sh`, `tasks/notify.sh`.

## test/

- Integration test docs/configs: `test/README.md`, `test/bot_integration_test.go`, `test/jsfull/conf/robot.yaml`.
