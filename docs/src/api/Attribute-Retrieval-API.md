# Getting Information About Users and the Robot

The attribute methods give plugins and jobs access to directory-style metadata about the current sender, another user, or the robot itself.

## Methods

- `GetSenderAttribute(attr)`
- `GetUserAttribute(user, attr)`
- `GetBotAttribute(attr)`

All three methods return both a value and a return code. In Go that is represented by `*robot.AttrRet`. In the scripting bindings, the object usually stringifies to the attribute value while still exposing the return code.

## Common user attributes

- `name`
- `fullName`
- `email`
- `firstName`
- `lastName`
- `phone`
- `internalID`

## Common bot attributes

- `name`
- `alias`
- `fullName` or `realName`
- `contact`, `admin`, or `adminContact`
- `email`
- `protocol`
- `internalID`

Most bot attributes come from robot configuration. User attributes come from the roster plus connector-local identity mapping.

## Examples

### Go
```go
func handler(r robot.Robot, args ...string) robot.TaskRetVal {
    sender := r.GetSenderAttribute("email")
    if sender.RetVal != robot.Ok {
        r.Say("I could not find your email address.")
        return robot.Normal
    }

    proto := r.GetBotAttribute("protocol")
    r.Say("I know you as %s, and I am speaking on %s.", sender.Attribute, proto.Attribute)
    return robot.Normal
}
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local ret = gopherbot.ret
local Robot = gopherbot.Robot

local bot = Robot:new()
local email, rv = bot:GetSenderAttribute("email")
if rv ~= ret.Ok then
  bot:Say("I could not find your email address.")
else
  local proto = bot:GetBotAttribute("protocol")
  bot:Say("I know you as " .. email .. " on " .. tostring(proto))
end
```

### JavaScript
```javascript
const { Robot, ret } = require("gopherbot_v1")();

const bot = new Robot();
const email = bot.GetSenderAttribute("email");
if (email.retVal !== ret.Ok) {
  bot.Say("I could not find your email address.");
} else {
  const proto = bot.GetBotAttribute("protocol");
  bot.Say(`I know you as ${email.attribute} on ${proto.attribute}.`);
}
```

### Bash
```bash
USEREMAIL=$(GetSenderAttribute email)
if [ $? -ne $GBRET_Ok ]
then
  Say "I could not find your email address."
else
  PROTOCOL=$(GetBotAttribute protocol)
  Say "I know you as $USEREMAIL on $PROTOCOL."
fi
```

### Python
```python
email = bot.GetSenderAttribute("email")
if email.ret != Robot.Ok:
    bot.Say("I could not find your email address.")
else:
    proto = bot.GetBotAttribute("protocol")
    bot.Say("I know you as %s on %s." % (email, proto))
```

### Ruby
```ruby
email = bot.GetSenderAttribute("email")
if email.ret != Robot::Ok
  bot.Say("I could not find your email address.")
else
  proto = bot.GetBotAttribute("protocol")
  bot.Say("I know you as #{email} on #{proto}.")
end
```
