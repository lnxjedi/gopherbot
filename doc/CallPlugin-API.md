# Calling Other Plugins
To make plugin code more modular and reusable, Gopherbot supports the ability for external plugins to call one another via the `CallPlugin(...)` method. `CallPlugin` returns a `PlugRetVal`, just as `Authorizer` and `Elevator` plugins. In practice, plugins will generally exit with `Normal` (0) for success, or `!=Normal` to indicate some kind of failure. (*Note:* `Authorizer` and `Elevator` plugins need to exit with status `Success` (7) to indicate success; library-defined constants should be used for this).

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

## PowerShell
```powershell
$status = $bot.CallPlugin("psdemo", @("power"))
if ( $status -ne "Normal" ) {
  $bot.Reply("Looks like there was a problem!")
}
```
**NOTE:** PowerShell class methods don't take a variable number of arguments, so the command and any arguments to a command need to be passed as a string array variable.


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
