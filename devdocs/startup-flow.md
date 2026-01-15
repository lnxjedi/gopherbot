# Gopherbot Startup Flow

This document details how Gopherbot starts up, detects its operating mode, and loads configuration.

## Overview

Gopherbot's startup is a multi-phase process:

1. **CLI parsing** - Process command-line flags
2. **Mode detection** - Determine startup mode from environment and filesystem
3. **Encryption initialization** - Set up brain encryption
4. **Pre-connect config load** - Load basic configuration without running scripts
5. **Brain initialization** - Start the brain provider
6. **Connector startup** - Start the chat protocol connector
7. **Post-connect config load** - Full configuration with plugin initialization

## Entry Points

- `main.go` → `bot.Start()` in `bot/start.go`
- `bot.Start()` → `initBot()` → `run()` in `bot/bot_process.go`

## Mode Detection

### Where: `bot/config_load.go:131` - `detectStartupMode()`

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
        // No custom repo - check for answerfile
        if _, err := os.Stat("answerfile.txt"); err == nil {
            return "setup"
        } else if _, ok := lookupEnv("ANS_PROTOCOL"); ok {
            return "setup"  // Container-based setup
        }
        return "demo"  // No config at all
    }

    // 4. Has GOPHER_CUSTOM_REPOSITORY - check if config exists
    robotYamlFile := filepath.Join(configPath, "conf", "robot.yaml")
    if _, err := os.Stat(robotYamlFile); err != nil {
        return "bootstrap"  // Need to clone config
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

| Mode | Conditions | Purpose |
|------|------------|---------|
| `cli` | `cliOp` flag set | Running a CLI command, not a robot |
| `test-dev` | `conf/robot.yaml` exists, not in `custom/` dir | Integration testing |
| `demo` | No config, no `GOPHER_CUSTOM_REPOSITORY`, no answerfile | Try the default robot (Floyd) |
| `setup` | `answerfile.txt` exists OR `ANS_*` env vars | Process setup wizard |
| `bootstrap` | `GOPHER_CUSTOM_REPOSITORY` set but no config yet | Clone custom config repo |
| `ide` | `GOPHER_IDE` env var set | Local development |
| `ide-override` | `GOPHER_IDE` set + override flag | IDE but connect to real chat |
| `production` | Config exists, not IDE | Normal operation |

## Configuration Template Processing

### Where: `conf/robot.yaml`

The default robot.yaml uses Go templates to set configuration based on mode:

```yaml
{{- $mode := GetStartupMode }}
{{- $proto := env "GOPHER_PROTOCOL" | default "terminal" }}
{{- $brain := env "GOPHER_BRAIN" | default "file" }}
{{- $logdest := env "GOPHER_LOGDEST" | default "stdout" }}

## Mode-specific overrides
{{- if eq $mode "demo" }}
  {{- $proto = "terminal" }}
  {{- $brain = "mem" }}
  {{- $logdest = "robot.log" }}
{{- else if eq $mode "bootstrap" }}
  {{- $proto = "nullconn" }}
  {{- $brain = "mem" }}
  {{- $logdest = "stdout" }}
{{- else if or (eq $mode "ide") (eq $mode "test-dev") }}
  {{- $proto = "terminal" }}
  {{- $brain = "mem" }}
  {{- $logdest = "robot.log" }}
{{- end }}

## Terminal should never log to stdout (interferes with UI)
{{- if and (eq $proto "terminal") (eq $logdest "stdout") }}
  {{- $logdest = "robot.log" }}
{{- end }}
```

### Template Functions

| Function | Purpose | Example |
|----------|---------|---------|
| `GetStartupMode` | Returns current mode string | `{{- $mode := GetStartupMode }}` |
| `env "VAR"` | Read environment variable | `{{ env "GOPHER_PROTOCOL" }}` |
| `default "val"` | Provide default if empty | `{{ env "X" \| default "y" }}` |
| `decrypt "..."` | Decrypt an encrypted value | `{{ decrypt "base64..." }}` |
| `.Include "file"` | Include another YAML file | `{{ .Include "slack.yaml" }}` |
| `SetEnv "VAR" "val"` | Override env for custom config | `{{ SetEnv "GOPHER_BRAIN" "mem" }}` |

## Encryption Initialization

### Where: `bot/bot_process.go:172-194`

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

### `initCrypt()` Flow (`bot/bot_process.go:254-316`)

1. Look for `GOPHER_ENCRYPTION_KEY` in environment
2. If found, try to decrypt the binary key file (`binary-encrypted-key[.environment]`)
3. If key file doesn't exist but env key does, generate new binary key
4. If no env key, check for legacy `EncryptionKey` in config (old style)

## Configuration Loading

### Two-Phase Loading

**Pre-connect load** (`loadConfig(true)`):
- Basic configuration only
- No external scripts run
- No plugin configuration loaded
- Happens before connector starts

**Post-connect load** (`loadConfig(false)`):
- Full configuration
- External plugin configs loaded (calls "configure" command)
- Scheduled jobs registered
- Plugins initialized (calls "init" command)

### Config Merge Order

1. Default config (`$GOPHER_INSTALLDIR/conf/robot.yaml`)
2. Custom config (`$GOPHER_CONFIGDIR/conf/robot.yaml`) - merges/overrides

## Plugin/Job Loading Based on Mode

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

### Where: `bot/bot_process.go:321-366` - `run()`

```
run()
  │
  ├─> go runBrain()           // Brain event loop (single goroutine)
  │
  ├─> go featureScan()        // Initial feature extraction
  │
  ├─> go sigHandle()          // Signal handler (SIGINT, SIGTERM, etc.)
  │
  └─> go conn.Run()           // Connector main loop
        │
        └─> (after connector ready)
              │
              ├─> loadConfig(false)    // Full config load
              └─> Log("Robot is initialized and running")
```

## Common Startup Issues

### "encryption not initialized"
- **Cause**: Mode requires encryption but no `GOPHER_ENCRYPTION_KEY` set
- **Fix**: Set the env var, or ensure you're in a mode that creates temp keys (demo/setup/ide)

### Plugins not loading
- **Cause**: Mode-based conditionals in robot.yaml excluding them
- **Check**: What mode is detected? Does the conditional match?

### Logs going to terminal
- **Cause**: `LogDest` set to `stdout` with terminal connector
- **Fix**: Automatic now (terminal forces `robot.log`), but can override in custom config

### go-bootstrap running when it shouldn't
- **Cause**: Mode detection returning wrong value
- **Check**: Is `GOPHER_CUSTOM_REPOSITORY` set? Does config exist?

## Key Files

- `bot/start.go` - Entry point, CLI parsing, log setup
- `bot/bot_process.go` - `initBot()`, `run()`, encryption init
- `bot/config_load.go` - `detectStartupMode()`, `loadConfig()`
- `conf/robot.yaml` - Default config with template logic

## Debugging Startup

1. Check what mode is detected:
   ```go
   Log(robot.Info, "Startup mode: %s", detectStartupMode())
   ```

2. Check template variable values by adding to robot.yaml:
   ```yaml
   ## Debug: {{ $mode }} / {{ $proto }} / {{ $logdest }}
   ```

3. Run with debug logging:
   ```bash
   GOPHER_LOGLEVEL=debug ./gopherbot
   ```
