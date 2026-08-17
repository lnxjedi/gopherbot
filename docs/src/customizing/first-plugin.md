# Your First Plugin

For a first plugin in v3, use Lua. It gives you a fast local loop without requiring an external runtime.

## 1. Register the plugin in `custom/conf/robot.yaml`

```yaml
ExternalPlugins:
  "hello":
    Description: Example hello-world plugin
    Path: plugins/hello.lua
```

## 2. Add matcher and help metadata in `custom/conf/plugins/hello.yaml`

For ordinary directed commands, use `SimpleMatcher`. See the [SimpleMatcher reference](../reference/SimpleMatcher.md) when you need captures, choices, options, or detailed matching rules.

```yaml
Commands:
- Command: hello
  SimpleMatcher: "hello"
  Usage: "hello"
  Summary: "reply with a friendly greeting"
  Examples:
  - "(alias) hello"
  Keywords: [ "hello", "example" ]
```

## 3. Create `custom/plugins/hello.lua`

```lua
local gopherbot = require "gopherbot_v1"
local Robot = gopherbot.Robot
local task = gopherbot.task

local cmd = arg and arg[1] or ""

if cmd == "init" then
  return task.Normal
end

if cmd == "hello" then
  local bot = Robot:new()
  bot:Reply("Hello from Lua")
  return task.Normal
end

return task.Fail
```

## 4. Reload and test

```text
;reload
;hello
```

## Why this layout matters

- command metadata lives in config, not buried in code
- the plugin code stays small and focused
- help output stays aligned with the actual command surface
