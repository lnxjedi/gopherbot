# Prompting for Replies

The `Prompt*ForReply` family supports interactive, multi-step commands where the robot needs more input before it can continue.

## Prompting methods

- `PromptForReply(regexID, prompt)`
- `PromptThreadForReply(regexID, prompt)`
- `PromptUserForReply(regexID, user, prompt)`
- `PromptUserChannelForReply(regexID, user, channel, prompt)`
- `PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt)`

The method you choose depends on who should receive the prompt and whether the exchange should stay inside a thread.

## Matching replies

`regexID` can refer to a custom reply matcher from plugin or job configuration, or one of the stock matchers:

- `Email`
- `Domain`
- `OTP`
- `IPAddr`
- `SimpleString`
- `YesNo`

## Return values

In Go, the methods return `(reply, retVal)`.

In the scripting bindings, the result is usually an object that includes the matched reply plus the return code.

Common return codes:

- `Ok`
- `UserNotFound`
- `ChannelNotFound`
- `MatcherNotFound`
- `Interrupted`
- `TimeoutExpired`
- `UseDefaultValue`
- `ReplyNotMatched`

## Examples

### Go
```go
func handler(r robot.Robot, args ...string) robot.TaskRetVal {
    answer, rv := r.PromptForReply("YesNo", "Do you want to continue?")
    if rv != robot.Ok {
        r.Say("No reply received.")
        return robot.Normal
    }
    r.Say("You answered: %s", answer)
    return robot.Normal
}
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local ret = gopherbot.ret
local Robot = gopherbot.Robot

local bot = Robot:new()
local answer, rv = bot:PromptForReply("YesNo", "Do you want to continue?")
if rv ~= ret.Ok then
  bot:Say("No reply received.")
else
  bot:Say("You answered: " .. answer)
end
```

### JavaScript
```javascript
const { Robot, ret } = require("gopherbot_v1")();

const bot = new Robot();
const answer = bot.PromptForReply("YesNo", "Do you want to continue?");
if (answer.retVal !== ret.Ok) {
  bot.Say("No reply received.");
} else {
  bot.Say(`You answered: ${answer.reply}`);
}
```

### Bash
```bash
ANSWER=$(PromptForReply "YesNo" "Do you want to continue?")
if [ $? -ne $GBRET_Ok ]
then
  Say "No reply received."
else
  Say "You answered: $ANSWER"
fi
```

### Python
```python
answer = bot.PromptForReply("YesNo", "Do you want to continue?")
if answer.ret != Robot.Ok:
    bot.Say("No reply received.")
else:
    bot.Say("You answered: %s" % answer)
```

### Ruby
```ruby
answer = bot.PromptForReply("YesNo", "Do you want to continue?")
if answer.ret != Robot::Ok
  bot.Say("No reply received.")
else
  bot.Say("You answered: #{answer}")
end
```
