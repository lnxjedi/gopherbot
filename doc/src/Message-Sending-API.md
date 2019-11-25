Table of Contents
=================

  * [Message Formatting](#message-formatting)
  * [Say and Reply](#say-and-reply)
  * [SendUserMessage, SendChannelMessage and SendUserChannelMessage](#sendusermessage-sendchannelmessage-and-senduserchannelmessage)
  * [Code Examples](#code-examples)
    * [Bash](#bash)
    * [PowerShell](#powershell)
    * [Python](#python)
    * [Ruby](#ruby)

# MessageFormat method and Message Formatting
**Gopherbot** is designed to provide ChatOps functionality for a variety of team
chat platforms, with Slack being the first. Due to the technical nature of ChatOps,
characters like `_`, `*` and ` may need to be rendered in replies to the user;
at times omitting these characters (because they cause formatting changes) could
remove important information. To provide the plugin author with the most flexibility,
**Gopherbot** supports the notion of three message formats:
* `Raw` - text sent by a plugin with `Raw` format is passed straight to the chat platform as-is; this is the default if no other default is specified
* `Variable` - for the `Variable` format, the protocol connector should attempt to process the message so that special characters are escaped or otherwise modified to render for the user in a standard variable-width font; for Slack, special characters are surrounded by nulls
* `Fixed` - the protocol connector should render `Fixed` format messages in a fixed-width block format

The `MessageFormat(raw|variable|fixed)` method returns a robot object with the specified format. A plugin can use
`GetBotAttribute("protocol")` to determine the connector protocol (e.g. "slack") to make intelligent decisions
about the format to use, or modify the content of raw messages depending on the connection protocol.

# Say and Reply
`Say` and `Reply` are the staples of message sending. Both are generally used for replying to the person who spoke to the robot, but `Reply` will also _mention_ the user. Normally, `Say` is used when the robot responds immediately to the user, but `Reply` is used when the robot is performing a task that takes more than a few minutes, and the robot needs to direct the message to the user to update them with progress on the task. Both `Say` and `Reply` take a `message` argument, and an optional second `format` argument that can be `variable` (the default) for variable-width text, or `fixed` for fixed-width text. The `fixed` format is normally used with embedded newlines to create tabular output where the columns will line up. The return value is not normally checked, but can be one of `Ok`, `UserNotFound`, `ChannelNotFound`, or `FailedMessageSend`.

# SendUserMessage, SendChannelMessage and SendUserChannelMessage
`Say` and `Reply` are actually convenience wrappers for the `Send*Message` family of methods. `SendChannelMessage` takes the obvious arguments of `channel` and `message` and just writes a message to a channel. `SendUserMessage` sends a direct message to a user, and `SendUserChannelMessage` directs the message to a user in a channel by using a connector-specific _mention_. Like `Say` and `Reply`, each of these functions also takes an optional `format` argument, and uses the same return values.

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

## PowerShell
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
