# Calling Other Plugins
To make plugin code more modular and reusable, Gopherbot supports the ability for one external plugin to call another via the `CallPlugin(...)` method. `CallPlugin` returns a `PlugRetVal`, just as `Authorizer` and `Elevator` plugins return a `PlugRetVal`.

The arguments to `CallPlugin` are:
 * The name of the plugin to call
 * The command to issue to the plugin
 * A variable number of arguments to the command being called

# Code Examples
## Bash
```bash
CallPlugin rubydemo recall
STATUS=$?
if [ "$STATUS" -ne "$PLUGRET_Normal" ]
then
	Say "Dang, there was a problem calling the rubydemo recall command: $STATUS"
fi
```

## Powershell
```powershell
$bot.Say("(not yet implemented)")
```

## Python
```python
status = bot.CallPlugin("echo", "echo", sys.argv.pop(0))
if status != Robot.Normal:
    bot.Say("Uh-oh, there was a problem calling the echo plugin, status: %d" % status)
```

## Ruby
```ruby
status = bot.CallPlugin("echo", "echo", ARGV[0])
if status != Robot::Normal
  bot.Say("Sorry, there was a problem calling the echo plugin, status: #{status}")
end
```
