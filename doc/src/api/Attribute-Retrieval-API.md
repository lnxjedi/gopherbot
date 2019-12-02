# Getting Information About Users and the Robot
The `Get*Attribute(...)` family of methods can be used to get basic chat service directory information like first and last name, email address, etc. `GetSenderAttribute` and `GetBotAttribute` take a single argument, the name of the attribute to retrieve. The lesser-used `GetUserAttribute` takes two arguments, the user and the attribute. The return value is an object with `Attribute` and `RetVal` members. `RetVal` will be one of `Ok`, `UserNotFound` or `AttributeNotFound`.

## User Attributes
The available attributes for a user / sender:
 * name (handle)
 * fullName
 * email
 * firstName
 * lastName
 * phone
 * internalID (protocol internal representatation)

## Bot Attributes
The available attributes for the bot:
 * name
 * alias
 * fullName / realName
 * contact / admin / adminContact
 * email
 * protocol (e.g. "slack")

Note: the values for most of these are configured in `conf/gopherbot.yaml`

# Code Examples
## Bash
```bash
USEREMAIL=$(GetSenderAttribute email)
if [ $? -ne $GBRET_Ok ]
then
  Say "I was unable to look up your email address"
else
  Say "Your email address is $USEREMAIL"
fi
```

## PowerShell
```powershell
$attr = $bot.GetBotAttribute("email")
if ( $attr.Ret -eq "Ok" ) {
  $email = $attr.Attr
  $bot.Say("My email address is: $email")
} else {
  $bot.Say("I don't think I have an email address")
}
```

## Python
```python
# In some cases you might forego error checking
bot.Say("You can send email to %s" % bot.GetBotAttribte("email"))
botNameAttr = bot.GetBotAttribute("fullName")
if botNameAttr.ret == Robot.Ok:
  bot.Say("My full name is %s" % botNameAttr)
else:
  bot.Say("I don't even know what my name is!")
```

## Ruby
```ruby
# In some cases you might forego error checking
bot.Say("You can send email to #{bot.GetBotAttribute("email")}")
botNameAttr = bot.GetBotAttribute("fullName")
if botNameAttr.ret == Robot::Ok
  bot.Say("My full name is #{botNameAttr}")
else
  bot.Say("I don't even know what my name is!")
end
```
