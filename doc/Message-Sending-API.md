Table of Contents
=================

  * [Say and Reply](#say-and-reply)
  * [SendUserMessage, SendChannelMessage and SendUserChannelMessage](#sendusermessage-sendchannelmessage-and-senduserchannelmessage)
  * [Message Formatting](#message-formatting)
  * [Code Examples](#code-examples)
    * [Bash](#bash)
    * [Powershell](#powershell)
    * [Python](#python)
    * [Ruby](#ruby)

# Say and Reply
`Say` and `Reply` are the staples of message sending. Both are generally used for replying to the person who spoke to the robot, but `Reply` will also _mention_ the user. Normally, `Say` is used when the robot responds immediately to the user, but `Reply` is used when the robot is performing a task that takes more than a few minutes, and the robot needs to direct the message to the user to update them with progress on the task. Both `Say` and `Reply` take a `message` argument, and an optional second `format` argument that can be `variable` (the default) for variable-width text, or `fixed` for fixed-width text. The `fixed` format is normally used with embedded newlines to create tabular output where the columns will line up. The return value is not normally checked, but can be one of `Ok`, `UserNotFound`, `ChannelNotFound`, or `FailedUserDM`.

# SendUserMessage, SendChannelMessage and SendUserChannelMessage
`Say` and `Reply` are actually convenience wrappers for the `Send*Message` family of methods. `SendChannelMessage` takes the obvious arguments of `channel` and `message` and just writes a message to a channel. `SendUserMessage` sends a direct message to a user, and `SendUserChannelMessage` directs the message to a user in a channel by using a connector-specific _mention_. Like `Say` and `Reply`, each of these functions also takes an optional `format` argument, and uses the same return values.

# Message Formatting
For the "fixed" and "variable" formats, Gopherbot processes each outgoing message to:
* Escape special characters like `&`, `<`, and `>` to make sure they're sent to the user as-is
* Ensure that "@<username>" creates a platform-specific "mention", e.g. Say("Tell @bofh hello")

# Code Examples
## Bash
```bash
# Note that bash isn't object-oriented
Say "I'm sending a message to Bob in #general" 
SendUserChannelMessage "bob" "general" "Hi, Bob!"
RETVAL = $?
if [ $RETVAL -ne $GBRET_Ok ]
then
  Log "Error" "Unable to message Bob in #general - return code $RETVAL"
fi
```

## Powershell
```powershell
$bot.Say("I'm sending a message to Bob in #general")
$retval = $bot.SendUserChannelMessage("bob", "general", "Hi, Bob!")
if ( $retval -ne "Ok" ) {
  $bot.Log("Error", "Unable to message Bob in #general - return code $retval")
}
```

## Python
```python
bot.Say("I'm sending a message to Bob in #general")
retval = bot.SendUserChannelMessage("bob", "general", "Hi, Bob!")
if ( retval != Robot.Ok ):
  bot.Log("Error", "Unable to message Bob in #general - return code %d" % retval)
```

## Ruby
```ruby
bot.Say("I'm sending a message to Bob in #general")
retval = bot.SendUserChannelMessage("bob", "general", "Hi, Bob!")
if retval != Robot::Ok
  bot.Log("Error", "Unable to message Bob in #general - return code %d" % retval)
end
```
