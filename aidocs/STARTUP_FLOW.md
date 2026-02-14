# Gopherbot Startup Flow

This document is intended as a **control-flow trace**, not a conceptual or tutorial overview. Its purpose is to make startup behavior explicit and verifiable for contributors working on the core engine (human or AI).

## Overview

Startup proceeds through the following phases **in order**:

1. **CLI parsing** – Process command-line flags
2. **Mode detection** – Determine startup mode from environment and filesystem
3. **Encryption initialization** – Set up brain encryption
4. **Pre-connect configuration load** – Load basic configuration without running scripts
5. **Brain initialization** – Start the brain provider
6. **Connector runtime initialization** – Initialize primary + configured secondary connectors
7. **Post-connect configuration load** – Full configuration with plugin initialization

## Entry Points

* `main.go` → `bot.Start()` in `bot/start.go`
* `bot.Start()` → `initBot()` → `run()` in `bot/bot_process.go`

CLI note:

- `--aidev <token>` enables AI development mode for the process (used by MCP automation flows).

## Mode Detection

### Where: `bot/config_load.go` – `detectStartupMode()`

The startup mode determines protocol, brain, logging, and which plugins load.

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
        // No custom repo – check for answerfile
        if _, err := os.Stat("answerfile.txt"); err == nil {
            return "setup"
        } else if _, ok := lookupEnv("ANS_PROTOCOL"); ok {
            return "setup" // Container-based setup
        }
        return "demo" // No config at all
    }

    // 4. Has GOPHER_CUSTOM_REPOSITORY – check if config exists
    robotYamlFile := filepath.Join(configPath, "conf", "robot.yaml")
    if _, err := os.Stat(robotYamlFile); err != nil {
        return "bootstrap" // Need to clone config
    }

    // 5. IDE mode checks
    if ideMode {
        if overrideIDEMode {
            return "ide-override"
        }
        return "ide"
    }

    return "production"
}
```

### Mode Summary

| Mode           | Conditions                                              | Purpose                                    |
| -------------- | ------------------------------------------------------- | ------------------------------------------ |
| `cli`          | `cliOp` flag set                                        | Running a CLI command, not a robot         |
| `test-dev`     | `conf/robot.yaml` exists, not in `custom/` dir          | Integration testing and engine development |
| `demo`         | No config, no `GOPHER_CUSTOM_REPOSITORY`, no answerfile | Run the default demo robot                 |
| `setup`        | `answerfile.txt` exists OR `ANS_*` env vars             | Process setup wizard                       |
| `bootstrap`    | `GOPHER_CUSTOM_REPOSITORY` set but no config yet        | Clone custom config repo                   |
| `ide`          | `GOPHER_IDE` env var set                                | Local development                          |
| `ide-override` | IDE mode with override flag                             | IDE but connect to real chat               |
| `production`   | Config exists, not IDE                                  | Normal operation                           |

## Robot Identity and Bootstrap Model

For contributors, "robot" is best understood as:

- A running Gopherbot process
- Backed by one custom configuration repository (`GOPHER_CUSTOM_REPOSITORY`)
- Initialized from environment (`.env` or process environment) and config templates

Bootstrap path (first configured start):

1. Startup mode resolves to `bootstrap` when `GOPHER_CUSTOM_REPOSITORY` is set but local config is absent (`detectStartupMode` in `bot/config_load.go`).
2. Default config selects `nullconn` for bootstrap mode and schedules `go-bootstrap` at `@init` (see `conf/robot.yaml`).
3. `go-bootstrap` (`gojobs/go-bootstrap/go_bootstrap_job.go`) validates required parameters (notably `GOPHER_CUSTOM_REPOSITORY`, `GOPHER_DEPLOY_KEY`), clones the custom repo, and queues `restart-robot`.
4. Process restarts and startup mode becomes `production` (or `ide`/`ide-override` depending on env).

Connector config implication:

- Installed defaults under `gopherbot/conf/` include only stock connector templates shipped with the engine.
- Connectors like Slack are normally configured in the custom robot repository's `conf/` and merged through custom `conf/robot.yaml` includes.

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
{{- else if or (eq $mode "ide") (eq $mode "test-dev") }}
  {{- if eq $mode "test-dev" }}
    {{- if IsTestBuild }}
  {{- $proto = "terminal" }}
    {{- else }}
  {{- $proto = "ssh" }}
    {{- end }}
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

### Protocol Key Compatibility

`bot/conf.go` now accepts both:

- `PrimaryProtocol` (preferred)
- `Protocol` (legacy alias for backward compatibility)

If both are set and differ, `PrimaryProtocol` wins and a warning is logged.

### Primary Protocol Config Source

Primary connector configuration now has explicit source precedence:

- Compatibility path: if `ProtocolConfig` is present in `robot.yaml`, it is used for the primary protocol and a warning is logged.
- Preferred path: if `ProtocolConfig` is absent in `robot.yaml`, engine loads `conf/<PrimaryProtocol>.yaml` and requires `ProtocolConfig` there.
- If preferred-path primary config file is missing or has no `ProtocolConfig`, startup/reload config load fails.

### Identity Mapping Key Compatibility

Identity mapping is explicit per-protocol `UserMap`.

- Preferred: `UserMap` (username -> protocol internal ID), typically in `conf/<protocol>.yaml`.
- `UserRoster` remains the user directory (email/name/phone/etc.).
- Legacy compatibility: `UserRoster.UserID` is still parsed to populate missing protocol mappings, with migration warnings.
- Precedence: explicit `UserMap` entries override legacy `UserRoster.UserID` entries on conflict.
- When primary config is auto-loaded from `conf/<primary>.yaml`, that file's `UserRoster` is ignored; attributes still come from `robot.yaml` `UserRoster`.
- For secondary protocol files, `UserRoster` attributes are ignored; use main `robot.yaml` `UserRoster` + protocol `UserMap`.
- With `IgnoreUnlistedUsers: true`, inbound users must satisfy both checks:
  - present in global `UserRoster` directory
  - mapped in protocol-specific identity mapping (`UserMap`)

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
  - each requires `Authorization: Bearer <token>`

This mode is additive: connector startup and config merge ordering are unchanged.

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
        // demo, setup, ide, test-dev: create temporary key
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
2. If found, try to decrypt the binary key file (`binary-encrypted-key[.environment]`)
3. If key file does not exist but env key does, generate a new binary key
4. If no env key exists, `initCrypt` finishes. A final check for a legacy `EncryptionKey` in `robot.yaml` is performed by `initBot()` after `initCrypt()` returns.

## Configuration Loading

**Invariant:** connector runtime startup must complete (primary required, secondaries best effort) before post-connect configuration is loaded.

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
## go-bootstrap only runs when NOT in demo/setup mode
{{- if not (or (eq $mode "demo") (eq $mode "setup")) }}
ScheduledJobs:
- Name: go-bootstrap
  Schedule: "@init"
{{- end }}

## welcome and autosetup only load in demo/setup mode
{{- if or (eq $mode "demo") (eq $mode "setup") }}
  {{- if eq $proto "terminal" }}
  "welcome":
    Path: plugins/welcome.lua
  {{- end }}
  "autosetup":
    Path: plugins/autosetup.sh
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
