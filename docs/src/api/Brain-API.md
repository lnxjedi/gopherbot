# Memory and Brain Methods

The Robot API has two different memory layers:

- long-term data stored in the configured brain provider
- short-term contextual memory stored in the running engine

## Memory scoping

Long-term data is scoped by key and namespace. Short-term memory is scoped to the current user and channel, and optionally to a thread.

## Long-term memory methods

- `CheckoutDatum(key, rw)`
- `UpdateDatum(...)`
- `CheckinDatum(...)`
- `DeleteDatum(key)`

Use long-term memory for persistent state such as deployment records, saved lists, or structured workflow state. Read-write checkouts are intentionally short-lived. If you need to coordinate longer work, combine long-term memory with `Exclusive(...)`.

Long-term memory helpers are not available in the Bash binding.

### Go
```go
type DeployState struct {
    Version string `json:"version"`
}

var state DeployState
token, exists, rv := r.CheckoutDatum("deploy-state", &state, true)
if rv != robot.Ok {
    r.Say("Unable to load deploy state.")
    return robot.Normal
}
if !exists {
    state.Version = "initial"
}
state.Version = "2026.03.17"
if rv := r.UpdateDatum("deploy-state", token, &state); rv != robot.Ok {
    r.Say("Unable to update deploy state.")
}
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local ret = gopherbot.ret
local Robot = gopherbot.Robot

local bot = Robot:new()
local memory, rv = bot:CheckoutDatum("deploy-state", true)
if rv ~= ret.Ok then
  bot:Say("Unable to load deploy state.")
elseif not memory.exists then
  memory.datum = { version = "initial" }
end

memory.datum.version = "2026.03.17"
if bot:UpdateDatum(memory) ~= ret.Ok then
  bot:Say("Unable to update deploy state.")
end
```

### JavaScript
```javascript
const { Robot, ret } = require("gopherbot_v1")();

const bot = new Robot();
const state = bot.CheckoutDatum("deploy-state", true);
if (state.retVal !== ret.Ok) {
  bot.Say("Unable to load deploy state.");
} else {
  if (!state.exists) {
    state.datum = { version: "initial" };
  }
  state.datum.version = "2026.03.17";
  if (bot.UpdateDatum(state) !== ret.Ok) {
    bot.Say("Unable to update deploy state.");
  }
}
```

### Python
```python
memory = bot.CheckoutDatum("deploy-state", True)
if memory.ret != Robot.Ok:
    bot.Say("Unable to load deploy state.")
else:
    if not memory.exists:
        memory.datum = {"version": "initial"}
    memory.datum["version"] = "2026.03.17"
    if bot.UpdateDatum(memory) != Robot.Ok:
        bot.Say("Unable to update deploy state.")
```

### Ruby
```ruby
memory = bot.CheckoutDatum("deploy-state", true)
if memory.ret != Robot::Ok
  bot.Say("Unable to load deploy state.")
else
  memory.datum = { "version" => "initial" } unless memory.exists
  memory.datum["version"] = "2026.03.17"
  if bot.UpdateDatum(memory) != Robot::Ok
    bot.Say("Unable to update deploy state.")
  end
end
```

## Short-term memory methods

- `Remember(key, value, shared)`
- `RememberThread(key, value, shared)`
- `RememberContext(context, value)`
- `RememberContextThread(context, value)`
- `Recall(key, shared)`
- `DeleteMemory(key, shared)`

Short-term memory is best for conversational context such as “the current server” or “the thing we were just talking about.”

### Go
```go
r.RememberContext("server", "api-01")
r.Say("Next time you say 'it', I will treat that as api-01.")
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local Robot = gopherbot.Robot

local bot = Robot:new()
bot:RememberContext("server", "api-01")
bot:Say("Next time you say 'it', I will treat that as api-01.")
```

### JavaScript
```javascript
const { Robot } = require("gopherbot_v1")();

const bot = new Robot();
bot.RememberContext("server", "api-01");
bot.Say("Next time you say 'it', I will treat that as api-01.");
```

### Bash
```bash
RememberContext "server" "api-01"
Say "Next time you say 'it', I will treat that as api-01."
```

### Python
```python
bot.RememberContext("server", "api-01")
bot.Say("Next time you say 'it', I will treat that as api-01.")
```

### Ruby
```ruby
bot.RememberContext("server", "api-01")
bot.Say("Next time you say 'it', I will treat that as api-01.")
```
