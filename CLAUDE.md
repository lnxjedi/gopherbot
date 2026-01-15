# CLAUDE.md - Gopherbot v3 Development Guide

This document provides context for AI assistants working on the Gopherbot project.

## Project Overview

**Gopherbot** is a DevOps chatbot framework written in Go. It connects to team chat platforms (Slack, Rocket.Chat, terminal) and executes plugins, jobs, and tasks in response to user messages or schedules. Think of it as a ChatOps alternative to Jenkins - users can write custom automation in any language.

**Key differentiator**: External scripts communicate with the robot over a localhost JSON/HTTP socket, allowing plugins in any language (bash, python, ruby, Go, etc.) that can source a simple library file.

## Repository Structure

```
gopherbot/
├── bot/                    # Core bot engine (Go)
│   ├── start.go           # Startup, CLI, bootstrap detection
│   ├── bot_process.go     # Main loops, goroutine spawning
│   ├── handler.go         # Message dispatch, worker spawning
│   ├── brain.go           # Serialized memory access (single goroutine)
│   ├── config_load.go     # YAML config loading with Go templates
│   └── run_pipelines.go   # Pipeline execution
├── robot/                  # Public robot interface (separate Go module)
├── connectors/             # Protocol connectors (slack, terminal, test)
├── brains/                 # Memory backends (dynamodb, cloudflarekv)
├── modules/                # Built-in interpreters
│   ├── javascript/        # Goja-based JS interpreter (v3)
│   ├── lua/               # gopher-lua interpreter (v3)
│   └── yaegi-dynamic-go/  # Yaegi Go interpreter
├── lib/                    # Client libraries for external scripts
│   ├── gopherbot_v1.sh    # Bash library
│   ├── gopherbot_v1.rb    # Ruby library
│   ├── gopherbot_v1.py    # Python library
│   ├── gopherbot_v1.js    # JavaScript library (v3)
│   └── gopherbot_v1.lua   # Lua library (v3)
├── plugins/                # Built-in plugins (mostly shell, converting to JS/Lua)
├── jobs/                   # Built-in jobs
├── tasks/                  # Built-in tasks
├── gojobs/                 # Go-native jobs (go-bootstrap, etc.)
├── gotasks/                # Go-native tasks (git-command, ssh-agent, etc.)
├── goplugins/              # Go-native plugins (groups, help, ping)
├── conf/                   # Default configuration (robot.yaml, etc.)
├── robot.skel/             # Skeleton for new robot repositories
├── resources/              # Answerfiles for setup
└── test/                   # Integration tests
```

## Concurrent Architecture

```
Main Process
    │
    ├─> go runBrain()          # Single goroutine serializing ALL memory ops
    │                          # (prevents race conditions)
    │
    ├─> go sigHandle()         # Signal handler
    │
    └─> go conn.Run()          # Connector loop (Slack, terminal, etc.)
            │
            └─> per message: go w.handleMessage()  # Worker per message
                    │
                    └─> startPipeline()  # Tasks run sequentially within pipeline
                            │
                            └─> external tasks: go callTaskThread()
                                    ├─> go stdout scanner
                                    └─> go stderr scanner
```

**Key insight**: The brain is a single goroutine with channel-based event loop - all memory operations are serialized through it, allowing many concurrent pipelines safely.

## Custom Robot Bootstrap Flow

When `GOPHER_CUSTOM_REPOSITORY` is set but no `custom/conf/robot.yaml` exists:

1. Startup uses `nullconn` (silent connector)
2. `go-bootstrap` job runs at `@init` schedule
3. Pipeline: ssh-agent → ssh-git-helper → git-command clone → restart-robot
4. On restart, finds `custom/conf/robot.yaml` and runs normally

See `gojobs/go-bootstrap/` and `bot/start.go:213-245` for implementation.

## Current Branch: v3_wip

We are working on the **v3_wip** branch. Key v3 goals:

### Completed in v3

1. **JavaScript interpreter** (goja) - Full Robot API (~1000 lines in lib/gopherbot_v1.js)
   - modules/javascript/ - 14 Go files implementing bindings
   - HTTP client for API calls (http_object.go)

2. **Lua interpreter** (gopher-lua) - Full Robot API (~685 lines in lib/gopherbot_v1.lua)
   - modules/lua/ - 14 Go files implementing bindings
   - Missing: HTTP client (parity needed with JS)

3. **CloudFlare KV brain** - Eventually-consistent design for slow backends
   - RAM cache + async write queue + flusher goroutine
   - See brains/cloudflarekv/cloudflarekvbrain.go

4. **Enhanced test infrastructure** - test/ directory improvements

### v3 Goals (TODO)

**Primary Goal**: Smooth bootstrap without external dependencies. A user should be able to:
```
/path/to/gopherbot    # in empty directory
;setup slack          # interactive prompts
```
...and get a working robot without needing bash, jq, git, python pre-installed.

### Priority 1: Bootstrap Flow Scripts (BLOCKING)

| Script | Lines | Status | Notes |
|--------|-------|--------|-------|
| `plugins/welcome.sh` | 42 | TODO | Trivial - just messaging |
| `plugins/autosetup.sh` | 358 | TODO | Main challenge - full setup wizard |
| `plugins/addadmin.sh` | ~100 | TODO | Adding initial admin |

**autosetup.sh blockers** - needs interpreter enhancements:
- `openssl rand` → need crypto/rand exposed to JS/Lua
- `ssh-keygen` → need Go-based SSH key generation task
- `git` operations → already have `git-command` Go task
- `sed` substitutions → need file read/write in JS/Lua

### Priority 2: Interpreter Enhancements

1. **Add HTTP client to Lua** (parity with JS http_object.go)
2. **Crypto functions** - random bytes generation for encryption keys
3. **SSH key generation** - Go task or interpreter function
4. **File I/O** - read/write/substitute for config manipulation

### Priority 3: Core Scripts Conversion

| Script | Purpose | Priority |
|--------|---------|----------|
| `jobs/save.sh` | Save config to git | Keep |
| `jobs/backup.sh` | Git-based brain backup | DEPRECATED |
| `jobs/restore.sh` | Git-based brain restore | DEPRECATED |
| `tasks/notify.sh`, `reply.sh`, `status.sh` | Simple messaging | Convert |

**Note**: backup/restore are deprecated. CloudFlare KV (and DynamoDB) provide persistent memory without git ugliness.

### Priority 4: Documentation

The documentation at `../gopherbot-doc/` needs updates for v3:
- New JS/Lua interpreters and APIs
- Removal of git-based backup/restore
- CloudFlare KV as recommended persistent brain
- Smoother bootstrap process
- Updated setup flow

Documentation uses mdbook format. Key files:
- `doc/src/SUMMARY.md` - Table of contents
- `doc/src/Installation.md`, `RobotSetup.md` - Setup docs
- `doc/src/api/` - API reference (needs JS/Lua examples)

## Build and Test

```bash
make                    # Build gopherbot binary
make test              # Run tests
./gopherbot            # Run from empty dir for bootstrap experience
./gopherbot -t         # Test mode with terminal connector
```

## Key Files for Understanding the Codebase

- `bot/start.go` - Startup flow, GOPHER_CUSTOM_REPOSITORY handling
- `bot/bot_process.go:314-359` - Main goroutine spawning
- `bot/handler.go:290` - Message dispatch, worker creation
- `modules/javascript/bot_object.go` - JS interpreter Robot binding
- `modules/lua/bot_userdata.go` - Lua interpreter Robot binding
- `lib/gopherbot_v1.js` - Full JS client API
- `lib/gopherbot_v1.lua` - Full Lua client API
- `plugins/autosetup.sh` - Current setup flow (needs conversion)

## Related Repositories

- `../clu-gopherbot/` - Primary development/test robot (named after Flynn's program from TRON)
- `../gopherbot-doc/` - Documentation (mdbook format)

## Dependencies

Current direct dependencies (go.mod):
- `github.com/dop251/goja` - JavaScript interpreter
- `github.com/yuin/gopher-lua` - Lua interpreter
- `github.com/traefik/yaegi` - Go interpreter
- `github.com/slack-go/slack` - Slack connector (via fork)
- `github.com/go-git/go-git/v5` - Git operations
- `github.com/aws/aws-sdk-go` - DynamoDB brain

Some dependencies are 2+ years old and should be updated as part of v3 work.
