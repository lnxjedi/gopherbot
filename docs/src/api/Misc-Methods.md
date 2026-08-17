# Miscellaneous Methods

This page covers public Robot methods that do not fit cleanly into the main attribute, messaging, memory, pipeline, or utility chapters.

## Thread subscriptions

These methods let a plugin stay attached to an active thread so later messages in that thread continue to route back to the same plugin logic.

- `Subscribe()`
- `Unsubscribe()`

### Go
```go
func handler(r robot.Robot, args ...string) robot.TaskRetVal {
    if r.Subscribe() {
        r.Say("I will keep watching this thread.")
    }
    return robot.Normal
}
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local Robot = gopherbot.Robot

local bot = Robot:new()
if bot:Subscribe() then
  bot:Say("I will keep watching this thread.")
end
```

### JavaScript
```javascript
const { Robot } = require("gopherbot_v1")();

const bot = new Robot();
if (bot.Subscribe()) {
  bot.Say("I will keep watching this thread.");
}
```

### Bash
```bash
if Subscribe
then
  Say "I will keep watching this thread."
fi
```

### Python
```python
if bot.Subscribe():
    bot.Say("I will keep watching this thread.")
```

### Ruby
```ruby
if bot.Subscribe()
  bot.Say("I will keep watching this thread.")
end
```

## Random helpers

These are convenience methods for lightweight reply variation.

- `RandomString(...)`
- `RandomInt(...)`

`RandomInt(...)` is not exposed consistently in every older scripting binding, so check the binding you are using.

### Go
```go
choice := r.RandomString([]string{"Working on it.", "Still running.", "Almost done."})
r.Say(choice)
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local Robot = gopherbot.Robot

local bot = Robot:new()
local choice = bot:RandomString({"Working on it.", "Still running.", "Almost done."})
bot:Say(choice)
```

### JavaScript
```javascript
const { Robot } = require("gopherbot_v1")();

const bot = new Robot();
const choice = bot.RandomString(["Working on it.", "Still running.", "Almost done."]);
bot.Say(choice);
```

### Python
```python
choice = bot.RandomString(["Working on it.", "Still running.", "Almost done."])
bot.Say(choice)
```

### Ruby
```ruby
choice = bot.RandomString(["Working on it.", "Still running.", "Almost done."])
bot.Say(choice)
```

## Compiled Go notes

Two public methods are mostly relevant to compiled Go extensions:

- `GetMessage()`
- `RaisePriv(reason)`

They are intentionally not expanded into large user-facing chapters here. If you are writing a compiled Go extension and need direct access to message internals or privileged filesystem behavior, read the engine source and current developer docs alongside this manual.
