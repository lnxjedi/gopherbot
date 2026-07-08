# Gopherbot Interpreters

This document describes how Gopherbot supports plugins, jobs, and tasks written in various programming languages.

## Two Categories of Interpreters

Gopherbot supports scripts in two fundamentally different ways:

### 1. Built-in Interpreters (Engine-managed runtimes)

Scripts are executed by Gopherbot-managed runtimes rather than by user-installed language helpers. In the current multiprocess execution model, interpreter-backed extensions run through `gopherbot pipeline-child-rpc`; Robot API calls go back to the parent engine over internal JSON RPC instead of the legacy HTTP helper path.

| Language | Runtime | Module Path | Child RPC methods | Notes |
|----------|---------|-------------|-------------------|-------|
| **Lua** | [gopher-lua](https://github.com/yuin/gopher-lua) | `modules/lua/` | `lua_run`, `lua_get_config` | Uses `lib/gopherbot_v1.lua` wrapper |
| **JavaScript** | [goja](https://github.com/dop251/goja) | `modules/javascript/` | `js_run`, `js_get_config` | Uses `lib/gopherbot_v1.js` wrapper |
| **Gopherbot shell** | [mvdan/sh](https://mvdan.cc/sh/) | `modules/gsh/` | `gsh_run`, `gsh_get_config` | Uses embedded `gopherbot_v1.gsh`; shell utilities stay inside the child |
| **Go** | [yaegi](https://github.com/traefik/yaegi) | `modules/yaegi-dynamic-go/` | `go_plugin_run`, `go_job_run`, `go_task_run`, `go_get_config` | Uses the Go `robot.Robot` API via RPC bridge |

**Advantages:**
- No user-managed external interpreter dependency
- Engine-controlled configure/init lifecycle
- Parent keeps authorization/routing/identity authority
- `.gsh` can expose a shell-style builtin utility surface without HTTP helper scripts

**File extensions:** `.lua`, `.js`, `.gsh`, `.go`

See `aidocs/EXTENSION_API.md` for the per-language Robot method surface and parity notes.

### 2. External Interpreters (JSON over HTTP)

Scripts run as separate processes. Robot API calls are **HTTP POST requests** to a localhost JSON endpoint.

| Language | Library File | Notes |
|----------|--------------|-------|
| **Bash** | `lib/gopherbot_v1.sh` | Uses curl for HTTP |
| **Python 2** | `lib/gopherbot_v1.py` | Uses urllib2 |
| **Python 3** | `lib/gopherbot_v2.py` | Uses urllib.request |
| **Ruby** | `lib/gopherbot_v1.rb` | Uses net/http |
| **Julia** | `lib/GopherbotV1.jl` | Experimental |

**How it works:**
1. Robot spawns script as subprocess
2. Script receives `GOPHER_HTTP_POST` environment variable (e.g., `http://127.0.0.1:35479`)
3. Script sources/imports the library
4. API calls become JSON POST to `$GOPHER_HTTP_POST/json`
5. Robot's HTTP handler (`bot/http.go`, method `ServeHTTP` on type `handler`) processes requests

**File extensions:** Any executable (`.sh`, `.py`, `.rb`, etc.)

## Invocation Flow

### Where: `bot/calltask.go` (funcs `callTask`, `callTaskThread`)

```
callTask()
    │
    ├─> Is .go file?
    │     └─> yaegi.RunPluginHandler() / RunJobHandler() / RunTaskHandler()
    │
    ├─> Is .lua file?
    │     └─> lua.CallExtension()
    │
    ├─> Is .js file?
    │     └─> js.CallExtension()
    │
    ├─> Is .gsh file?
    │     └─> gsh.CallExtension()
    │
    └─> Otherwise (external script)
          └─> exec.Command() with GOPHER_HTTP_POST in environment
```

Built-in interpreter children and external executable scripts receive the same
working-directory behavior. `Homed: true` starts execution in the robot home
directory; otherwise execution starts in the current pipeline working
directory, including changes made through `SetWorkingDirectory`.

Environment behavior is intentionally split between host/user values and robot
values:

- `HOME` and `PATH` are inherited from the parent process environment when set.
- `GOPHER_HOME` is the robot home/root directory.
- `GOPHER_CONFIGDIR`, `GOPHER_INSTALLDIR`, `GOPHER_WORKSPACE`, and other
  `GOPHER_*` values describe robot runtime context.
- Ruby external scripts receive `GEM_HOME` pointing at
  `$GOPHER_HOME/.bot-gems`. This directory is the robot-managed Ruby dependency
  cache; the shipped `install-libs` job installs Bundler and then runs Bundler
  with `BUNDLE_PATH__SYSTEM=true`, so Gemfile dependencies are installed as
  normal RubyGems under `GEM_HOME`. The engine does not override `GEM_PATH`, so
  system gems remain visible to external Ruby scripts.
- Python external scripts receive `PYTHONUSERBASE` pointing at
  `$GOPHER_HOME/.bot-python`. The shipped `install-libs` job installs
  `requirements.txt` dependencies there with `pip install --user` and makes
  dependency code readable/traversable by unprivileged privsep children.

Extension authors should use normal `HOME`/`PATH` semantics for host tools such
as `kubectl`, `git`, and cloud CLIs, and use `GOPHER_HOME` when they mean the
robot directory.

## Local Development CLI

`gopherbot syntax` and `gopherbot script` are no-init CLI commands for local
development of built-in interpreter scripts. They operate on explicit paths
resolved from the current working directory; they do not discover configured
plugins by name.

`gopherbot syntax [options] <script> [script...]` checks Lua, JavaScript, GSH,
and interpreted Go scripts without loading robot configuration. Language is
inferred from `.lua`, `.js`, `.gsh`, or `.go`, or supplied with
`-language lua|js|gsh|go`. Use `-json`/`-j` for structured diagnostics. GSH
syntax checking prepends the embedded `gopherbot_v1.gsh` shim before parsing so
the check matches runtime parsing.

`gopherbot script [options] <script> [--] <command> [args...]` runs one
built-in interpreter script through the same child RPC runtime used by normal
pipeline execution, but with a local fixture-backed Robot API instead of a live
connector/worker/brain. In plugin mode, the command after `--` is prepended
before capture args, matching configured plugin execution. In `-kind job` or
`-kind task`, args are passed as-is. `--` is recommended whenever the plugin
command or captures may look like CLI flags.

Useful options:

- `-c <source>` runs inline source and requires `-language` or
  `fixture.language`.
- `-fixture <path>` loads JSON message/config/parameter/memory/prompt data.
- `-no-interactive` makes prompts return `TimeoutExpired` when fixture replies
  are exhausted; by default prompts fall back to reading one line from stdin.
- `-json` records messages, prompts, logs, memory updates, and pipeline API
  calls as structured events.

The local Robot API is intentionally conservative:

- `Say`, `Reply`, `SendChannelMessage`, and related methods write readable
  terminal lines such as `#general @alice: message`.
- Prompt methods write prompts such as `#general @alice Name of cat?: ` and
  consume `fixture.prompts.replies` before falling back to stdin.
- `GetTaskConfig`, `GetParameter`, message/user/bot attributes, short-term
  memory, and datum memory are served from the fixture/local in-memory state.
- Cloud-bound or privileged mechanisms such as shared encryption and identity
  linking return normal failure codes unless fixture data explicitly supplies a
  read path such as `identities`.

Runnable examples and fixture data live under `test-scripts/`; see
`test-scripts/README.md`.

## Plugin Contract by Language

Config key note: in v3 plugin config, directed command matchers must be under `Commands`. Legacy `CommandMatchers` and top-level `Help` are no longer accepted.
Directed `Commands` may use either `Regex` (raw Go regex) or `SimpleMatcher` (the simpler command DSL compiled to regex by the engine). `MessageMatchers` remain regex-only.
SimpleMatcher captures arrive positionally: typed slots (`<name:type>`), required labelled capturing choices (`(label:...)` or `(:...)`), and optional labelled capturing groups (`[label:...]` or `[:...]`) become args; argv-style options blocks (`[-label:-flag|-name:<type>]`) emit one arg per matched option and therefore make following captures variable-position; plain text, required synonyms (`/.../`), and optional noise (`{...}`) do not. The detailed contract lives in `devdocs/SimpleMatcher.md`; the engine diagnostic design lives in `aidocs/SIMPLE_MATCHER_DIAGNOSTICS.md`.

Engine-owned plugin callbacks always use command names beginning with `_`.
Configured plugin `Commands` and `MessageMatchers` must not use that prefix.
Current engine callbacks include `_configure`, `_init`, `_authorize`,
`_usergroups`, `_elevate`, `_catchall`, `_subscribed`, and `_expiresub`.

### Lua Plugins

**Entry point:** Script is executed directly; check `arg[1]` for command.

```lua
-- plugins/example.lua
local robot = require("gopherbot_v1")

local command = arg[1]

if command == "_configure" then
    -- MUST return a string (YAML config or empty "")
    return [[
Commands:
- SimpleMatcher: example
  Command: example
]]
end

if command == "_init" then
    -- Initialization, return task code
    return robot.task.Normal
end

-- Handle user commands
local bot = robot.Robot:new()
if command == "example" then
    bot:Say("Hello from Lua!")
    return robot.task.Normal
end

return robot.task.Normal
```

**Key points:**
- `arg[1]` = command (`_configure`, `_init`, or user command)
- `arg[2]`, `arg[3]`, ... = capture groups from either `Regex` or `SimpleMatcher`
- `_configure` **must return a string** (empty `""` if no config)
- `GBOT` global provides the raw bot userdata
- Use `robot.Robot:new()` to get a wrapped Robot instance

**See:** `plugins/samples/hello.lua`, `plugins/samples/demo.lua`

### JavaScript Plugins

**Entry point:** Script is executed; check `argv[1]` for command.

```javascript
// plugins/example.js
const robot = require('gopherbot_v1');

const command = argv[1];

if (command === "_configure") {
    // Return YAML config string
    return `
Commands:
- Regex: '(?i:example)'
  Command: example
`;
}

if (command === "_init") {
    return robot.Normal;
}

const bot = new robot.Robot();
if (command === "example") {
    bot.Say("Hello from JavaScript!");
    return robot.Normal;
}

return robot.Normal;
```

**Key points:**
- `argv[1]` = command
- `argv[2]`, `argv[3]`, ... = regex capture groups
- `_configure` returns a string
- `GBOT` global provides the raw bot object
- Use `new robot.Robot()` for wrapped instance
- Has HTTP client for external API calls (`robot.http`)

**See:** `plugins/samples/hello.js`, `plugins/samples/demo.js`

### Dynamic Go Plugins (Yaegi)

**Entry point:** Must export `Configure()` and `PluginHandler()` functions.

```go
// plugins/example/example.go
package main

import "github.com/lnxjedi/gopherbot/robot"

var defaultConfig = []byte(`
Commands:
- Regex: '(?i:example)'
  Command: example
`)

func Configure() *[]byte {
    return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {
    switch command {
    case "_init":
        // initialization
    case "example":
        r.Say("Hello from Go!")
    }
    return robot.Normal
}
```

**Key points:**
- Package must be `main`
- `Configure() *[]byte` - returns pointer to YAML config bytes
- `PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal`
- For jobs: `JobHandler(r robot.Robot, args ...string) robot.TaskRetVal`
- For tasks: `TaskHandler(r robot.Robot, args ...string) robot.TaskRetVal`
- Direct access to `robot.Robot` interface

**See:** `plugins/go-knock/knock.go`, `plugins/go-lists/lists.go`

### Gopherbot Shell Plugins

**Entry point:** First argument is the dispatched command, just like a shell script, but Robot methods and common shell utilities are builtin commands provided by the integrated `.gsh` runtime.

```sh
#!/bin/sh

default_config() {
cat <<'EOF'
---
Commands:
- Regex: (?i:hello gsh)
  Command: hello
EOF
}

command=$1
shift

case "$command" in
  _configure)
    default_config
    ;;
  hello)
    say "Hello, Gopherbot shell World!"
    ;;
esac
```

**Key points:**
- `$1` is the command (`_configure`, `_init`, or the configured command name).
- Robot methods such as `say`, `Reply`, `PromptForReply`, `AddTask`, and `GetTaskConfig` are builtin shell commands, not HTTP wrappers.
- `Log` accepts either numeric levels (`0` Trace, `1` Debug, `2` Info, `3` Audit, `4` Warn, `5` Error, `6` Fatal) or named levels such as `Log Audit "Something happened"`; unknown named levels, including `Fatal`, log as `Error` for compatibility with external bash scripts.
- Common shell utilities are also builtin (`cat`, `cp`, `find`, `grep`, `jq`, `ls`, `mktemp`, `mv`, `sort`, `tar`, `touch`, `tr`, `uniq`, `wc`, `xargs`, and more).
- Builtin `jq` is powered by the `gojq` library and has a Gopherbot wrapper modeled on the upstream `gojq` CLI. GSH plugins can use normal jq language features such as `first(g)` plus common CLI features including `--arg`, `--argjson`, `$ARGS`, `--slurpfile`, `--rawfile`, `-r`, `-c`, `-n`, `-s`, `-R`, `-e`, `-f`, `-L`, `--stream`, and YAML input/output flags. The wrapper runs inside the GSH interpreter, so stdin/stdout/stderr, relative file paths, and exported environment variables come from the active GSH command context rather than process-global `os.Stdin` / `os.Environ`.
- Builtin `head` and `tail` accept both `-n <count>` and shell-compatible shorthand count flags like `-1`.
- Command lookup is case-insensitive across Robot builtins, so `say` and `Say` are equivalent.
- Maintained engine-shipped script defaults now prefer `.gsh` entrypoints (for example `plugins/admin.gsh`, `tasks/status.gsh`, and `tasks/notify.gsh`) while legacy `.sh` examples remain in-tree for compatibility/reference.

**See:** `plugins/samples/hello.gsh`, `plugins/test/shfull.gsh`

### External Scripts (Bash, Python, Ruby)

**Entry point:** First argument is command; source library for API access.

```bash
#!/bin/bash
# plugins/example.sh

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift

case "$command" in
    "_configure")
        cat << 'EOF'
Commands:
- Regex: '(?i:example)'
  Command: example
EOF
        ;;
    "_init")
        # initialization
        ;;
    "example")
        Say "Hello from Bash!"
        ;;
esac
```

```python
#!/usr/bin/env python3
# plugins/example.py

import os
import sys
sys.path.append(os.getenv('GOPHER_INSTALLDIR') + '/lib')
from gopherbot_v2 import Robot

bot = Robot()
command = sys.argv[1]

if command == "_configure":
    print("""
Commands:
- Regex: '(?i:example)'
  Command: example
""")
elif command == "_init":
    pass
elif command == "example":
    bot.Say("Hello from Python!")
```

```ruby
#!/usr/bin/env ruby
# plugins/example.rb

require ENV['GOPHER_INSTALLDIR'] + '/lib/gopherbot_v1'

bot = Robot.new
command = ARGV[0]

case command
when "_configure"
    puts <<~EOF
Commands:
- Regex: '(?i:example)'
  Command: example
EOF
when "_init"
    # initialization
when "example"
    bot.Say("Hello from Ruby!")
end
```

**Key points:**
- First argument (`$1`, `sys.argv[1]`, `ARGV[0]`) is command
- `_configure`: Print YAML to stdout, exit 0
- Library functions make HTTP calls to `$GOPHER_HTTP_POST/json`
- `GOPHER_INSTALLDIR` points to gopherbot installation

## Environment Variables Available to Scripts

All scripts receive these environment variables:

| Variable | Description |
|----------|-------------|
| `GOPHER_INSTALLDIR` | Path to gopherbot installation |
| `GOPHER_CONFIGDIR` | Path to custom config (if privileged/homed) |
| `GOPHER_WORKSPACE` | Workspace directory |
| `GOPHER_HTTP_POST` | HTTP endpoint for JSON API (external scripts only) |
| `GOPHER_CHANNEL` | Current channel name |
| `GOPHER_USER` | Current user name |
| `GOPHER_PROTOCOL` | Protocol name (slack, terminal, etc.) |
| `GOPHER_TASK_NAME` | Name of current task |
| `GOPHER_PIPELINE_TYPE` | "plugin", "job", or "task" |

Plus any parameters defined in the task configuration.

## Configuration Loading

### Built-in Interpreters

Configuration is loaded by calling the script with `_configure` argument:
- **Lua:** `modules/lua/get_config.go` - `GetPluginConfig()`
- **JavaScript:** `modules/javascript/get_config.go` - `GetPluginConfig()`
- **Yaegi:** Calls `Configure()` function directly

### External Scripts

Configuration loaded by:
1. Running script with `_configure` as first argument
2. Capturing stdout
3. Parsing as YAML

See: `bot/calltask.go` (func `getDefCfgThread`) for external script config loading.

## API Parity

The goal is for all interpreters to provide the same Robot API. Current status:

| Feature | Lua | JavaScript | Yaegi Go | External |
|---------|-----|------------|----------|----------|
| Messaging (Say, Reply, etc.) | ✓ | ✓ | ✓ | ✓ |
| Prompting | ✓ | ✓ | ✓ | ✓ |
| Long-term memory (brain) | ✓ | ✓ | ✓ | ✓ |
| Short-term memory | ✓ | ✓ | ✓ | ✓ |
| Pipeline control | ✓ | ✓ | ✓ | ✓ |
| Attributes | ✓ | ✓ | ✓ | ✓ |
| HTTP client | ✓ | ✓ | ✓ | ✓ |
| File I/O | ✗ | ✗ | ✓ | ✓ |

## Debugging Scripts

### Built-in Interpreters
- Errors appear in robot log
- Use `bot:Log(log.Debug, "message")` (Lua) or `bot.Log(robot.Debug, "message")` (JS)
- Script errors include stack traces

### External Scripts
- stderr goes to robot log with "ERR" prefix
- stdout during `_configure` is captured as config
- Use library Log functions for structured logging
- Check `robot.log` for HTTP request/response debugging

## Key Files

**Built-in interpreter modules:**
- `modules/lua/` - Lua interpreter (14 Go files)
- `modules/javascript/` - JavaScript interpreter (14 Go files)
- `modules/yaegi-dynamic-go/` - Yaegi Go interpreter

**Client libraries:**
- `lib/gopherbot_v1.lua` - Lua Robot API (~685 lines)
- `lib/gopherbot_v1.js` - JavaScript Robot API (~1000 lines)
- `lib/gopherbot_v1.sh` - Bash Robot API
- `lib/gopherbot_v1.py` - Python 2 Robot API
- `lib/gopherbot_v2.py` - Python 3 Robot API
- `lib/gopherbot_v1.rb` - Ruby Robot API
- `lib/GopherbotV1.jl` - Julia Robot API (experimental)

**Invocation:**
- `bot/calltask.go` - Main dispatch logic
- `bot/http.go` - JSON API handler for external scripts

**Sample plugins:**
- `plugins/samples/hello.lua`, `demo.lua` - Lua examples
- `plugins/samples/hello.js`, `demo.js` - JavaScript examples
- `plugins/go-knock/knock.go` - Dynamic Go example
- `plugins/samples/rubydemo.rb` - Ruby example
- `plugins/samples/pythondemo.py` - Python example
