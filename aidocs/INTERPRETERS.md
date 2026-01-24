# Gopherbot Interpreters

This document describes how Gopherbot supports plugins, jobs, and tasks written in various programming languages.

## Two Categories of Interpreters

Gopherbot supports scripts in two fundamentally different ways:

### 1. Built-in Interpreters (Direct Calls)

Scripts are executed within the Go process using embedded interpreters. Robot API calls are **direct function calls** - no network overhead.

| Language | Interpreter | Module Path | Library |
|----------|-------------|-------------|---------|
| **Lua** | [gopher-lua](https://github.com/yuin/gopher-lua) | `modules/lua/` | `lib/gopherbot_v1.lua` |
| **JavaScript** | [goja](https://github.com/dop251/goja) | `modules/javascript/` | `lib/gopherbot_v1.js` |
| **Go** | [yaegi](https://github.com/traefik/yaegi) | `modules/yaegi-dynamic-go/` | N/A (uses robot package) |

**Advantages:**
- No process spawning overhead
- No JSON serialization/deserialization
- Direct access to robot.Robot interface
- Faster execution

**File extensions:** `.lua`, `.js`, `.go`

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
    └─> Otherwise (external script)
          └─> exec.Command() with GOPHER_HTTP_POST in environment
```

## Plugin Contract by Language

### Lua Plugins

**Entry point:** Script is executed directly; check `arg[1]` for command.

```lua
-- plugins/example.lua
local robot = require("gopherbot_v1")

local command = arg[1]

if command == "configure" then
    -- MUST return a string (YAML config or empty "")
    return [[
Help:
- Keywords: [ "example" ]
  Helptext: [ "(bot), example - do something" ]
CommandMatchers:
- Regex: '(?i:example)'
  Command: example
]]
end

if command == "init" then
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
- `arg[1]` = command ("configure", "init", or user command)
- `arg[2]`, `arg[3]`, ... = regex capture groups
- `configure` **must return a string** (empty `""` if no config)
- `GBOT` global provides the raw bot userdata
- Use `robot.Robot:new()` to get a wrapped Robot instance

**See:** `plugins/samples/hello.lua`, `plugins/samples/demo.lua`

### JavaScript Plugins

**Entry point:** Script is executed; check `argv[1]` for command.

```javascript
// plugins/example.js
const robot = require('gopherbot_v1');

const command = argv[1];

if (command === "configure") {
    // Return YAML config string
    return `
Help:
- Keywords: [ "example" ]
  Helptext: [ "(bot), example - do something" ]
CommandMatchers:
- Regex: '(?i:example)'
  Command: example
`;
}

if (command === "init") {
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
- `configure` returns a string
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
Help:
- Keywords: [ "example" ]
  Helptext: [ "(bot), example - do something" ]
CommandMatchers:
- Regex: '(?i:example)'
  Command: example
`)

func Configure() *[]byte {
    return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {
    switch command {
    case "init":
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

### External Scripts (Bash, Python, Ruby)

**Entry point:** First argument is command; source library for API access.

```bash
#!/bin/bash
# plugins/example.sh

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift

case "$command" in
    "configure")
        cat << 'EOF'
Help:
- Keywords: [ "example" ]
  Helptext: [ "(bot), example - do something" ]
CommandMatchers:
- Regex: '(?i:example)'
  Command: example
EOF
        ;;
    "init")
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

if command == "configure":
    print("""
Help:
- Keywords: [ "example" ]
  Helptext: [ "(bot), example - do something" ]
CommandMatchers:
- Regex: '(?i:example)'
  Command: example
""")
elif command == "init":
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
when "configure"
    puts <<~EOF
Help:
- Keywords: [ "example" ]
  Helptext: [ "(bot), example - do something" ]
CommandMatchers:
- Regex: '(?i:example)'
  Command: example
EOF
when "init"
    # initialization
when "example"
    bot.Say("Hello from Ruby!")
end
```

**Key points:**
- First argument (`$1`, `sys.argv[1]`, `ARGV[0]`) is command
- `configure`: Print YAML to stdout, exit 0
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

Configuration is loaded by calling the script with "configure" argument:
- **Lua:** `modules/lua/get_config.go` - `GetPluginConfig()`
- **JavaScript:** `modules/javascript/get_config.go` - `GetPluginConfig()`
- **Yaegi:** Calls `Configure()` function directly

### External Scripts

Configuration loaded by:
1. Running script with "configure" as first argument
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
| HTTP client | ✗ | ✓ | ✓ | ✓ |
| File I/O | ✗ | ✗ | ✓ | ✓ |

**v3 TODO:** Add HTTP client to Lua for parity with JavaScript.

## Debugging Scripts

### Built-in Interpreters
- Errors appear in robot log
- Use `bot:Log(log.Debug, "message")` (Lua) or `bot.Log(robot.Debug, "message")` (JS)
- Script errors include stack traces

### External Scripts
- stderr goes to robot log with "ERR" prefix
- stdout during "configure" is captured as config
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
