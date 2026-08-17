# Utility Methods

This chapter covers the core convenience methods that most plugin and job authors use regularly.

## Methods

- `CheckAdmin()`
- `Elevate(immediate)`
- `GetTaskConfig(...)`
- `Log(level, message, ...)`
- `Pause(seconds)`

## Notes

- `CheckAdmin()` is useful when one plugin has a mix of admin-only and non-admin commands.
- `Elevate(...)` requests stronger proof of identity, usually for sensitive operations.
- `GetTaskConfig(...)` is how compiled Go, Lua, and JavaScript extensions read structured task configuration. Python and Ruby also expose it. Bash does not.
- `Log(...)` writes to the robot log with the normal log levels.

## Log levels

- `Trace`
- `Debug`
- `Info`
- `Audit`
- `Warn`
- `Error`
- `Fatal`

## Examples

### Go
```go
type config struct {
    AllowedEnv string `json:"allowedEnv"`
}

var cfg config
if rv := r.GetTaskConfig(&cfg); rv != robot.Ok {
    r.Log(robot.Error, "missing config")
    return robot.Normal
}
if !r.CheckAdmin() {
    r.Say("This command is admin-only.")
    return robot.Normal
}
r.Log(robot.Info, "running admin task for %s", cfg.AllowedEnv)
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local ret = gopherbot.ret
local task = gopherbot.task
local log = gopherbot.log
local Robot = gopherbot.Robot

local bot = Robot:new()
local cfg, rv = bot:GetTaskConfig()
if rv ~= ret.Ok then
  bot:Log(log.Error, "missing config")
  return task.Normal
end
if not bot:CheckAdmin() then
  bot:Say("This command is admin-only.")
  return task.Normal
end
bot:Log(log.Info, "running admin task")
```

### JavaScript
```javascript
const { Robot, ret, log } = require("gopherbot_v1")();

const bot = new Robot();
const cfg = bot.GetTaskConfig();
if (cfg.retVal !== ret.Ok) {
  bot.Log(log.Error, "missing config");
} else if (!bot.CheckAdmin()) {
  bot.Say("This command is admin-only.");
} else {
  bot.Log(log.Info, "running admin task");
}
```

### Bash
```bash
if ! CheckAdmin
then
  Say "This command is admin-only."
  exit 0
fi

Log "Info" "running admin task"
Pause 1
Say "Done."
```

### Python
```python
if not bot.CheckAdmin():
    bot.Say("This command is admin-only.")
else:
    bot.Log("Info", "running admin task")
    bot.Pause(1)
    bot.Say("Done.")
```

### Ruby
```ruby
if !bot.CheckAdmin()
  bot.Say("This command is admin-only.")
else
  bot.Log("Info", "running admin task")
  bot.Pause(1)
  bot.Say("Done.")
end
```
