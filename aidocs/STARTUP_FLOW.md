# Gopherbot Startup Flow

This document is intended as a **control-flow trace**, not a conceptual or tutorial overview. Its purpose is to make startup behavior explicit and verifiable for contributors working on the core engine (human or AI).

## Overview

Startup proceeds through the following phases **in order**:

1. **CLI parsing** – Process command-line flags
2. **Initial mode probe** – Evaluate startup mode for early IDE working-directory behavior
3. **Private environment load** – Load `private/environment` or `.env` when present
4. **Effective mode detection** – Re-evaluate startup mode using process env plus loaded private env
5. **Encryption initialization** – Set up brain encryption
6. **Pre-connect configuration load** – Load basic configuration without running scripts
7. **Brain initialization** – Start the brain provider
8. **Connector runtime initialization** – Initialize primary + configured secondary connectors
9. **Post-connect configuration load** – Full configuration with plugin initialization
10. **Runtime git branch capture** – Best-effort detection of current/default startup branch for admin observability

Internal exception:
- `pipeline-child-exec` is an internal command used by multiprocess task execution; it exits after one child-task run and bypasses normal robot startup phases.
- `pipeline-child-rpc` is an internal command used by multiprocess RPC execution; it runs a versioned stdio RPC loop (including Lua/JavaScript/Go execution and configure methods) and bypasses normal robot startup phases.

## Entry Points

* `main.go` → `bot.Start()` in `bot/start.go`
* `bot.Start()` → `initBot()` → `run()` in `bot/bot_process.go`

Test harness note:

- `StartTest()` in `bot/start_t.go` follows the same `initBot()` → `run()` path, then waits for the current async plugin-init batch to reach quiescence before returning to the integration harness. This test-only barrier does not change production startup behavior.

CLI note:

- `--aidev <token>` enables AI development mode for the process (used by MCP automation flows).

Internal child-runner note:

- `gopherbot pipeline-child-exec` is parsed immediately after flag parsing in `Start(...)`.
- `gopherbot pipeline-child-rpc` is parsed in the same early dispatch block in `Start(...)`.
- When either command is detected, startup calls the internal child path and returns without loading config, brain, connectors, or HTTP listeners.

## Mode Detection

### Where: `bot/config_load.go` – `detectStartupMode()`

The startup mode determines protocol, brain, logging, and which plugins load.
`Start(...)` in `bot/start.go` calls `detectStartupMode()` twice: an early probe before private env loading, then a second pass after private env loading that becomes the effective startup mode.

```go
func detectStartupMode() (mode string) {
    // 1. CLI operation (encrypt, decrypt, etc.)
    if cliOp {
        return "cli"
    }

    // 2. Test/development config exists in current directory
    if _, err := os.Stat(filepath.Join("conf", "robot.yaml")); err == nil {
        cwd, _ := os.Getwd()
        if !strings.HasSuffix(cwd, "/custom") {
            return "test-dev"
        }
    }

    // 3. Check if robot is "configured" (has GOPHER_CUSTOM_REPOSITORY)
    _, robotConfigured := lookupEnv("GOPHER_CUSTOM_REPOSITORY")
    if !robotConfigured {
        return "demo" // No config at all
    }

    // 4. Has GOPHER_CUSTOM_REPOSITORY – check if config exists
    robotYamlFile := filepath.Join(configPath, "conf", "robot.yaml")
    if _, err := os.Stat(robotYamlFile); err != nil {
        return "bootstrap" // Need to clone config
    }

    return "production"
}
```

### Mode Summary

| Mode           | Conditions                                              | Purpose                                    |
| -------------- | ------------------------------------------------------- | ------------------------------------------ |
| `cli`          | `cliOp` flag set                                        | Running a CLI command, not a robot         |
| `test-dev`     | `conf/robot.yaml` exists, not in `custom/` dir          | Integration testing and engine development |
| `demo`         | No config, no `GOPHER_CUSTOM_REPOSITORY`                | Run the default demo robot                 |
| `bootstrap`    | `GOPHER_CUSTOM_REPOSITORY` set but no config yet        | Clone custom config repo                   |
| `production`   | Config exists for configured robot                      | Normal operation                           |

## Robot Identity and Bootstrap Model

For contributors, "robot" is best understood as:

- A running Gopherbot process
- Backed by one custom configuration repository (`GOPHER_CUSTOM_REPOSITORY`)
- Initialized from environment (`.env` or process environment) and config templates

Bootstrap path (first configured start):

1. Startup mode resolves to `bootstrap` when `GOPHER_CUSTOM_REPOSITORY` is set but local config is absent (`detectStartupMode` in `bot/config_load.go`).
2. Default config selects `nullconn` for bootstrap mode and schedules `go-bootstrap` at `@init` (see `conf/robot.yaml`).
3. `go-bootstrap` (`gojobs/go-bootstrap/go_bootstrap_job.go`) validates required parameters (notably `GOPHER_CUSTOM_REPOSITORY`, `GOPHER_DEPLOY_KEY`), clones the custom repo, and queues `restart-robot`.
4. Process restarts and startup mode becomes `production`.

Connector config implication:

- Installed defaults under `gopherbot/conf/` include only stock connector templates shipped with the engine.
- Connectors like Slack are normally configured in the custom robot repository under `conf/protocols/`.

### Custom Robot Environment Selection (`GOPHER_ENVIRONMENT`)

For robots created from `robot.skel`, custom configuration is environment-driven:

- `custom/conf/robot.yaml` includes `conf/environments/<environment>.yaml`.
- `GOPHER_ENVIRONMENT` selects the environment file (default `development`).
- Environment files define runtime defaults for that robot environment (for example protocol, brain, log destination).

Representative custom robot template pattern:

```yaml
{{ $environment := env "GOPHER_ENVIRONMENT" | default "development" }}
{{ printf "environments/%s.yaml" $environment | .Include }}
```

This is distinct from installed engine defaults (`gopherbot/conf/robot.yaml`), which still use startup mode logic to bootstrap/demo/test behavior before or without a custom robot repository.

## Configuration Template Processing

### Where: `conf/robot.yaml`

The default `robot.yaml` uses Go templates to derive configuration values from startup mode and environment.

```yaml
{{- $mode := GetStartupMode }}
{{- $proto := env "GOPHER_PROTOCOL" | default "ssh" }}
{{- $brain := env "GOPHER_BRAIN" | default "file" }}
{{- $logdest := env "GOPHER_LOGDEST" | default "stdout" }}

## Mode-specific overrides
{{- if eq $mode "demo" }}
  {{- $proto = "ssh" }}
  {{- $brain = "mem" }}
  {{- $logdest = "stdout" }}
{{- else if eq $mode "bootstrap" }}
  {{- $proto = "nullconn" }}
  {{- $brain = "mem" }}
  {{- $logdest = "stdout" }}
{{- else if eq $mode "test-dev" }}
  {{- if IsTestBuild }}
  {{- $proto = "terminal" }}
  {{- else }}
  {{- $proto = "ssh" }}
  {{- end }}
  {{- $brain = "mem" }}
  {{- if and (eq $mode "test-dev") (eq $proto "terminal") }}
  {{- $logdest = "robot.log" }}
  {{- else }}
  {{- $logdest = "stdout" }}
  {{- end }}
{{- end }}

## Terminal should never log to stdout (interferes with UI)
{{- if and (eq $proto "terminal") (eq $logdest "stdout") }}
  {{- $logdest = "robot.log" }}
{{- end }}
```

### Protocol Selection Keys

`bot/conf.go` requires:

- `PrimaryProtocol` in `robot.yaml` (required)
- `DefaultProtocol` in `robot.yaml` (optional; defaults to `PrimaryProtocol`)

If `DefaultProtocol` is set, it must be the primary protocol or one of `SecondaryProtocols`; otherwise startup logs a warning and falls back to `PrimaryProtocol`.

### Default Outgoing Message Format

`DefaultMessageFormat` in `robot.yaml` is optional.

- Supported values: `Raw`, `Fixed`, `Variable`, `BasicMarkdown`
- If unset, startup defaults to `BasicMarkdown`
- Unknown values log an error and fall back to `BasicMarkdown`
- This default applies only when a plugin/job/built-in send path does not explicitly select a message format.
- Calls chained from `Robot.MessageFormat(...)` override the default for that send.

### Primary Protocol Config Source

Primary connector configuration is always loaded from:

- `conf/protocols/<PrimaryProtocol>.yaml`

`ProtocolConfig` is expected there (not in `robot.yaml`). If that file is missing, or missing `ProtocolConfig`, startup/reload config load fails.

### Connector Initialization Contract

Connector registration is static, but connector capabilities are resolved at initialization time.

- `robot.RegisterConnector(name, Initialize)` registers the connector type.
- During connector runtime startup, the engine calls `Initialize(...)` for each active protocol.
- `Initialize(...)` returns `robot.InitializedConnector{Connector, Capabilities}`.
- Zero-value `ConnectorCapabilities` means "no special connector capabilities".
- This allows capability flags like `HiddenCommands` to depend on protocol config instead of being fixed at registration time.
- Because pre-connect config is already loaded before connector runtime initialization, connectors can also consume shared robot identity at init time through `Handler.GetBotInfo()` without needing protocol-local bot-name duplicates.

Practical example:

- Slack now decides hidden-command support during `Initialize(...)` based on `ProtocolConfig.AcceptSlashCommands` and `ProtocolConfig.SlashCommand`.
- SSH/test/terminal currently return hidden-command support unconditionally from their initialized connector result.

### Brain/History Provider Config Sources

Provider-specific configuration is loaded by selected provider name:

- brain settings: `conf/brains/<Brain>.yaml` with top-level `BrainConfig`
- history settings: `conf/history/<HistoryProvider>.yaml` with top-level `HistoryConfig`

`BrainConfig` and `HistoryConfig` are invalid top-level keys in `robot.yaml`.

If a selected provider file is missing, or missing its required top-level key, startup/reload config load fails.

### Config Merge Semantics (Installed Defaults + Custom Overrides)

`bot/config_load.go` merges installed defaults with custom config using these rules:

- map values merge by key recursively
- scalar values are overridden by custom values
- slice values are replaced by custom values unless the key uses `Append*` prefix

Design implication:

- protocol identity config that must fully replace defaults should prefer list entries over map keys where practical
- SSH `ProtocolConfig.UserKeys` is a list of `{UserName, PublicKeys}` entries to avoid default-user key bleed-through during map merge
- to clear installed default SSH users in custom robots, set `ProtocolConfig.UserKeys: []` (or provide explicit entries)

### Engine-Shipped Extension Config Layering

For extensions shipped with Gopherbot (compiled or default external extensions):

- installed defaults are authoritative (`conf/plugins/*.yaml`, `conf/jobs/*.yaml`, extension `Configure()` defaults)
- custom robot extension config should be minimal and local: enable/disable, local parameters/secrets, and intentional local behavior deltas
- avoid copying full default command lists or policy lists (`Commands`, `AdminCommands`, `AuthorizedCommands`, etc.) into `custom/conf` unless behavior is intentionally being redefined
- prefer `Append*` keys when adding list entries to preserve upstream defaults and reduce drift

This keeps upgrade behavior predictable and prevents stale custom copies from disabling or diverging shipped extension behavior.

### Identity Mapping Key Compatibility

Identity policy is username-authoritative in engine flows.

- `UserRoster` is the global user directory (email/name/phone/etc.) and policy membership list.
- Inbound security identity uses connector-provided canonical username.
- `IgnoreUnlistedUsers: true` gates on username membership in global `UserRoster` (connector username trust boundary).
- Outbound engine-to-connector user sends are username-based; connectors resolve protocol-local IDs internally.
- Connector-reported bot IDs are stored per protocol (`protocol -> botID`) via `SetBotID(...)`.
- `GetBotAttribute("id")` resolves to:
  - the triggering protocol's bot ID for inbound plugin/message pipelines
  - `DefaultProtocol` bot ID for job/init/scheduled pipelines without inbound protocol context
  - no fallback to a legacy single global bot ID field
- Top-level `UserMap` is an invalid key in `robot.yaml` and protocol files (config load fails fast).
- Connector-local identity mapping belongs in each connector's `ProtocolConfig`.
- Slack identity mapping: `ProtocolConfig.UserMap`.
- SSH identity mapping: `ProtocolConfig.UserKeys` list entries (`UserName` + `PublicKeys`).
- Terminal/test identity mapping: connector-local `ProtocolConfig.Users` tables.
- `UserRoster.UserID` is accepted for config compatibility but ignored by engine identity policy.
- Ephemeral user-scoped memory keys are username-based (not `UserID`-based).
- Thread-scoped ephemeral memory keys include protocol context (`protocol + threadID`) to avoid cross-protocol thread key collisions.

`SecondaryProtocols` is accepted in `robot.yaml` and now drives active runtime orchestration:

- startup attempts the primary connector and all configured secondaries
- secondary startup failures are logged and do not abort startup
- `terminal` is not supported as a secondary protocol; if listed it is ignored with a warning
- reload reconciles secondary runtime (removed secondaries stop; configured secondaries are re-attempted)
- changing primary protocol on reload is rejected and logged; active primary remains unchanged

### AI Development Mode (`--aidev`)

When `--aidev <token>` is supplied at startup:

- startup forces `logDest` to `robot.log` (in the process working directory)
- after the local HTTP listener binds, startup writes the listener port to `<working-directory>/.aiport`
- the token is startup state for follow-on authenticated AI-dev endpoint work
- local authenticated endpoints are enabled on the existing localhost listener:
  - `POST /aidev/send_message`
  - `POST /aidev/get_messages`
  - `POST /aidev/send_as_robot`
  - each requires `Authorization: Bearer <token>`

This mode is additive: connector startup and config merge ordering are unchanged.

Test harness note:

- integration startup in `bot/start_t.go` waits for the current async plugin-init batch to quiesce before returning control to the harness, so startup `init` events do not bleed into the first assertion

### Template Functions

| Function             | Purpose                        | Example                             |
| -------------------- | ------------------------------ | ----------------------------------- |
| `GetStartupMode`     | Returns current mode string    | `{{- $mode := GetStartupMode }}`    |
| `env "VAR"`          | Read environment variable      | `{{ env "GOPHER_PROTOCOL" }}`       |
| `default "val"`      | Provide default if empty       | `{{ env "X" \| default "y" }}`      |
| `decrypt "..."`      | Decrypt an encrypted value     | `{{ decrypt "base64..." }}`         |
| `.Include "file"`    | Include another YAML file      | `{{ .Include "slack.yaml" }}`       |
| `SetEnv "VAR" "val"` | Override env for custom config | `{{ SetEnv "GOPHER_BRAIN" "mem" }}` |

## Encryption Initialization

### Where: `bot/bot_process.go` — within func `initBot()`, encryption initialization block

```go
encryptionInitialized := initCrypt()
if encryptionInitialized {
    setEnv("GOPHER_ENCRYPTION_INITIALIZED", "initialized")
} else {
    mode := detectStartupMode()
    switch mode {
    case "cli", "bootstrap", "production":
        // These modes REQUIRE encryption
        Log(robot.Fatal, "unable to initialize encryption...")
    default:
        // demo and test-dev: create temporary key
        bk := make([]byte, 32)
        crand.Read(bk)
        cryptKey.Lock()
        cryptKey.key = bk
        cryptKey.initialized = true
        cryptKey.Unlock()
    }
}
```

### `initCrypt()` Flow — `bot/bot_process.go` (func `initCrypt`)

1. Look for `GOPHER_ENCRYPTION_KEY` in environment
2. If found, resolve the binary key file path:
   - prefer `binary-encrypted-key.<environment>` when `GOPHER_ENVIRONMENT` is non-production and that file exists
   - otherwise fall back to `binary-encrypted-key`
3. If neither candidate file exists but env key does, generate a new binary key at the base path `binary-encrypted-key`
4. If no env key exists, `initCrypt` finishes. A final check for a legacy `EncryptionKey` in `robot.yaml` is performed by `initBot()` after `initCrypt()` returns.

`GOPHER_ENVIRONMENT` therefore has two related startup roles:
- selecting custom robot environment files under `custom/conf/environments/`
- optionally selecting a separate encrypted binary key file when an environment-specific key file is intentionally present

Operational note:
- this preserves easy reuse of the shared encrypted-secret domain in development by default
- operators can still opt into separate encrypted credentials for a given environment by creating `binary-encrypted-key.<environment>`

## Configuration Loading

**Invariant:** connector runtime startup must complete (primary required, secondaries best effort) before post-connect configuration is loaded.

After post-connect configuration load, startup performs a best-effort runtime git branch capture (`initializeRuntimeGitState` in `bot/git_runtime.go`). This records:
- current branch (`HEAD`)
- startup branch (branch active at process startup)
- default branch from local git metadata (`refs/remotes/origin/HEAD`, local-only, no network fallback)

This updates in-memory runtime state and `GOPHER_CUSTOM_BRANCH`-family internal env values for observability, but does not affect configuration precedence or startup mode decisions.

## Extension Loading

**Go extensions (compiled)** are registered at init time and wired before startup: `main.go` calls `bot.ProcessRegistrations()` before `bot.Start()`, which consumes registrations collected via `robot/registrations.go` (funcs `RegisterPlugin`, `RegisterJob`, `RegisterTask`).

**External extensions (scripts)** are discovered during config load: `conf/robot.yaml` defines `ExternalPlugins`, `ExternalJobs`, and `ExternalTasks` (see `bot/conf.go` fields), and `bot/taskconf.go` (func `addExternalTask`) converts them into runtime tasks during `loadConfig(true/false)`. Post-connect `loadConfig(false)` loads external plugin config (`configure`) and runs plugin init.

Related map: `aidocs/EXTENSION_SURFACES.md`.

### Two-Phase Loading

**Pre-connect load** (`loadConfig(true)`):

* Basic configuration only
* No external scripts run
* No plugin configuration loaded
* Occurs before connector startup

**Post-connect load** (`loadConfig(false)`):

* Full configuration
* External plugin configs loaded (calls `configure` command)
* Scheduled jobs registered
* Plugins initialized (calls `init` command)

### Config Merge Order

1. Default config (`$GOPHER_INSTALLDIR/conf/robot.yaml`)
2. Custom config (`$GOPHER_CONFIGDIR/conf/robot.yaml`) – merges and overrides

## Plugin and Job Loading Based on Mode

### Conditional Loading in `conf/robot.yaml`

```yaml
## go-bootstrap only runs when NOT in demo mode
{{- if ne $mode "demo" }}
ScheduledJobs:
- Name: go-bootstrap
  Schedule: "@init"
{{- end }}

## welcome/onboarding hooks only load in demo mode
{{- if eq $mode "demo" }}
  {{- if or (eq $proto "terminal") (eq $proto "ssh") }}
  "welcome":
    Path: plugins/welcome.lua
  "new-robot":
    Path: plugins/go-new-robot/new_robot.go
  {{- end }}
{{- end }}
```

## Goroutine Startup

### Where: `bot/bot_process.go` – `run()`

```
run()
  │
  ├─> go runBrain()        // Brain event loop (single goroutine)
  │
  ├─> go sigHandle()       // Signal handler (SIGINT, SIGTERM, etc.)
  │
  ├─> startConnectorRuntimes()
  │     ├─> primary connector run loop (required)
  │     └─> secondary connector run loops (best effort)
  │
  └─> loadConfig(false)    // Run full config load in the main thread
        │
        └─> Log("Robot is initialized and running")
```

## Runtime Protocol Controls

Built-in admin commands now expose protocol runtime lifecycle controls from the primary protocol:

- `protocol-list` / `protocol list`
- `protocol-start <name>` / `protocol start <name>`
- `protocol-stop <name>` / `protocol stop <name>`
- `protocol-restart <name>` / `protocol restart <name>`

These commands operate on configured secondary protocols only. Primary protocol changes remain startup-only.

## Shutdown Sequence (Control-Flow)

Shutdown can be triggered by admin commands, pipeline tasks, or process signals. The shutdown flow is:

1. Set `state.shuttingDown = true` (prevents new non-allowed pipelines).
2. Call `stop()` in `bot/bot_process.go`.
3. `stop()` first triggers prompt shutdown signaling so in-progress `Prompt*` waits return `Interrupted` immediately.
4. `stop()` waits for running pipelines (`state.Wait()`).
5. Stop brain loop (`brainQuit()`), then stop connector runtimes.
6. Stop signal handler goroutine.
7. Emit restart flag on `done` channel.

This keeps shutdown deterministic even when interactive prompts are using long timeout windows.

## Key Files

* `bot/start.go` – Entry point, CLI parsing, log setup
* `bot/bot_process.go` – `initBot()`, `run()`, encryption initialization
* `bot/privsep.go` – privilege-separation bootstrap + thread-scoped uid transitions
* `bot/aidev.go` – AI-dev startup state (`--aidev`) and `.aiport` write helper
* `bot/aidev_http.go` – authenticated AI-dev message endpoints and connector capability routing
* `bot/config_load.go` – `detectStartupMode()`, config-file merge/template expansion
* `bot/conf.go` – `loadConfig()` and reload reconciliation hooks for `SecondaryProtocols`
* `bot/connector_runtime.go` – multi-connector runtime manager, routing, lifecycle controls
* `conf/robot.yaml` – Default config with template logic

Related execution model reference: `aidocs/EXECUTION_SECURITY_MODEL.md`.

## Debugging Startup (Human-Oriented Notes)

1. Log detected startup mode:

   ```go
   Log(robot.Info, "Startup mode: %s", detectStartupMode())
   ```

2. Inspect template variable values by adding to `robot.yaml`:

   ```yaml
   ## Debug: {{ $mode }} / {{ $proto }} / {{ $logdest }}
   ```

3. Run with debug logging:

   ```bash
   GOPHER_LOGLEVEL=debug ./gopherbot
   ```
