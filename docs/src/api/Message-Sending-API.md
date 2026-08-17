# Sending Messages and Formatting

The Robot API separates message destination from message format. That makes it possible to keep most extensions protocol-agnostic while still sending readable output to Slack, SSH, and other connectors.

## Message formats

- `BasicMarkdown`
  - the v3 default outgoing format
  - intended to be the portable, cross-protocol choice for most normal replies
- `Variable`
  - literal variable-width text
  - useful when you want normal text without connector-driven formatting surprises
- `Fixed`
  - preformatted output for code, logs, and tabular text
- `Raw`
  - connector-native pass-through
  - use sparingly, because it gives up portability

For the actual `BasicMarkdown` syntax contract, see [BasicMarkdown Reference](../BasicMarkdown.md).

## Default and overrides

Outgoing format is chosen in this order:

1. an explicit format on the send call or helper Robot
2. `GOPHER_MESSAGE_FORMAT` if set
3. `DefaultMessageFormat`
4. the v3 default of `BasicMarkdown`

The usual helpers are:

- `MessageFormat(...)`
- `Fixed()`
- `Direct()`
- `Threaded()`

## Main message-sending methods

- `Say(...)` and `Reply(...)`
  - send in the current context
- `SayThread(...)` and `ReplyThread(...)`
  - force thread-aware delivery
- `SendChannelMessage(...)` and `SendChannelThreadMessage(...)`
  - send to a specific channel
- `SendUserMessage(...)`
  - send a direct message
- `SendUserChannelMessage(...)` and `SendUserChannelThreadMessage(...)`
  - direct a message to a user in a channel
- `SendProtocolUserChannelMessage(...)`
  - choose the connector explicitly for cross-protocol sends

## Platform-specific rendering

The API guarantees the intent of the format, not identical pixel-perfect rendering across every connector. For connector-specific notes such as Slack formatting behavior, check the relevant connector appendix after reading this page.

## Examples

### Go
```go
func handler(r robot.Robot, args ...string) robot.TaskRetVal {
    r.Say("Build started for `%s`", args[0])
    r.Fixed().Say("service   status\napi       ok\nworker    ok")
    r.Direct().Reply("I will DM you when the deploy finishes.")
    _ = r.SendProtocolUserChannelMessage("slack", "", "deployments", "*Deploy complete*")
    return robot.Normal
}
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local fmt = gopherbot.fmt
local Robot = gopherbot.Robot

local bot = Robot:new()
bot:MessageFormat(fmt.BasicMarkdown):Say("Deploying `payments`")
bot:Fixed():Say("service   status\napi       ok\nworker    ok")
bot:Direct():Reply("I will DM you when the deploy finishes.")
bot:SendProtocolUserChannelMessage("slack", "", "deployments", "*Deploy complete*", fmt.BasicMarkdown)
```

### JavaScript
```javascript
const { Robot, fmt } = require("gopherbot_v1")();

const bot = new Robot();
bot.MessageFormat(fmt.BasicMarkdown).Say("Deploying `payments`");
bot.Fixed().Say("service   status\napi       ok\nworker    ok");
bot.Direct().Reply("I will DM you when the deploy finishes.");
bot.SendProtocolUserChannelMessage("slack", "", "deployments", "*Deploy complete*", fmt.BasicMarkdown);
```

### Bash
```bash
Say -m 'Deploying `payments`'
Say -f $'service   status\napi       ok\nworker    ok'
SendUserMessage -m "$GOPHER_USER" "I will DM you when the deploy finishes."
SendProtocolUserChannelMessage -m "slack" "" "deployments" "*Deploy complete*"
```

### Python
```python
bot.MessageFormat("BasicMarkdown").Say("Deploying `payments`")
bot.MessageFormat("Fixed").Say("service   status\napi       ok\nworker    ok")
bot.Direct().Reply("I will DM you when the deploy finishes.")
bot.SendProtocolUserChannelMessage("slack", "", "deployments", "*Deploy complete*", "BasicMarkdown")
```

### Ruby
```ruby
bot.MessageFormat("BasicMarkdown").Say("Deploying `payments`")
bot.MessageFormat("Fixed").Say("service   status\napi       ok\nworker    ok")
bot.Direct().Reply("I will DM you when the deploy finishes.")
bot.SendProtocolUserChannelMessage("slack", "", "deployments", "*Deploy complete*", "BasicMarkdown")
```
