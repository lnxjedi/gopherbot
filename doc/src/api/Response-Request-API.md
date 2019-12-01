The `Prompt*ForReply` methods make it simple to write interactive plugins where the bot can request additional input from the user.

Table of Contents
=================

  * [Technical Background](#technical-background)
  * [Prompting Methods](#prompting-methods)
    * [Method Arguments](#method-arguments)
    * [Return Values](#return-values)
  * [Code Examples](#code-examples)
    * [Bash](#bash)
    * [PowerShell](#powershell)
    * [Python](#python)
    * [Ruby](#ruby)

## Technical Background

Interactive plugins are complicated by the fact that multiple plugins can be running simultaneously and each can request input from the user. **Gopherbot** handles requests for replies this way:

1. If there are no other plugins waiting for a reply for the given user/channel, the robot emits the prompt and waits to hear back from the user
2. If other plugins are waiting for a reply, the prompt is not emitted and the request goes in to a list of waiters
3. As other plugins get replies (or timeout while waiting), waiters in the list get a `RetVal` of `RetryPrompt`, indicating they should issue the prompt request again (this is handled internally in individual scripting libraries)

## Prompting Methods

The following methods are available for prompting for replies:
* `PromptForReply(regexID string, prompt string)` - issue a prompt to whoever/wherever the original command was issued
* `PromptUserForReply(regexID string, user string, prompt string)` - for prompting the user in a direct message (DM) (for e.g. a password or other sensitive information)
* `PromptUserChannelForReply(regexID string, user string, channel string, prompt string)` - prompt a specific user in a specific channel (for e.g. getting approval from another user for an action)

### Method arguments
The `user` and `channel` arguments are obvious; the `prompt` is the question the robot is
asking the user, and should usually end with a `?`.

The `regexID` should correspond to a `ReplyMatcher` defined in the plugin configuration,
(see [Plugin Configuration](Configuration.md#plugin-configuration)), or one of the
built-in regex's:
*	`Email`
* `Domain` - an alpha-numeric domain name
* `OTP` - a 6-digit one-time password code
* `IPAddr`
* `SimpleString` - Characters commonly found in most english sentences, doesn't include special characters like @, {, etc.
* `YesNo`

### Return Values

Two distinct values are returned from the prompting methods:
1. A `RetVal` indicating success or error condition - `Reply.ret`
2. When `RetVal` == `Ok`, the matched string is also returned - `Reply.reply`

In **Go**, these are returned as two separate values; in most scripting
languages, these are returned as a compound object whose string representation
is the returned string in `Reply.reply` (if the `RetVal` was `Ok`, otherwise it's the empty string).

Possible values for the `RetVal` in `Reply.ret` are:
* `Ok` - If the user replied and the reply matched the regex identified by `regexID`
* `UserNotFound`, `ChannelNotFound` - When an invalid user / channel is provided
* `MatcherNotFound` - When an invalid matcher is supplied
* `Interrupted` - If the user issues a new command to the robot (see NOTE below), too many `RetryPrompt` values are returned (>3), or the user replies with a single dash: '`-`' (cancel)
* `TimeoutExpired` - If the user says nothing for 45 seconds
* `UseDefaultValue` - If the user replied with a single equal sign (`=`)
* `ReplyNotMatched` - When the reply from the user didn't match the supplied regex (the user was probably talking to somebody else)

## Code Examples
### Bash
```bash
# Note that bash isn't object-oriented
REPLY=$(PromptForReply "YesNo" "Do you like kittens?")
if [ $? -ne 0 ]
then
	Reply "Eh, sorry bub, I'm having trouble hearing you - try typing faster?"
else
  if [[ $REPLY == y* ]] || [[ $REPLY == Y* ]]
  then
    Say "No kidding! Me too!"
  else
    Say "Oh, come on - you're kidding, right?!?"
  fi
fi
```

### PowerShell
```powershell
$rep = $bot.PromptForReply("YesNo", "Do you like kittens?")
if ($rep.Ret -ne "Ok") {
  $bot.Say("Eh, sorry bub, I'm having trouble hearing you - try typing faster?")
} else {
  $reply = [String]$rep
  switch -Wildcard ($reply) {
    "y*" { # PS is case-insensitive
      $bot.Say("No kidding! Me too!")
    }
    default {
      $bot.Say("Oh, come on - you're kidding, right?!?")
    }
  }
}
```

### Python
```python
rep = bot.PromptForReply("YesNo", "Do you like kittens?")
if rep.ret != Robot.Ok:
  bot.Say("Eh, sorry bub, I'm having trouble hearing you - try typing faster?")
else:
  reply = rep.__str__()
  if re.match("y.*", reply, flags=re.IGNORECASE):
    bot.Say("No kidding! Me too!")
  else:
    bot.Say("Oh, come on - you're kidding, right?!?")
```

### Ruby
```ruby
rep = bot.PromptForReply("YesNo", "Do you like kittens?")
if rep.ret != Robot::Ok
  bot.Say("Eh, sorry bub, I'm having trouble hearing you - try typing faster?")
else
  reply = rep.to_s()
  if /y.*/i =~ reply
    bot.Say("No kidding! Me too!")
  else
    bot.Say("Oh, come on - you're kidding, right?!?")
  end
end
```