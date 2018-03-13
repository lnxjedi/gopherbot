# Log Method

Besides the logging that Gopherbot does on it's own, plugins can also emit log messages with one of the following log levels:

 * `Trace` - for fine-grained logging of all actions
 * `Debug` - for emitting debugging info
 * `Info` - the default log level
 * `Audit` - for auditable events - **NOTE:** *Audit events are always logged regardless of the current log level*
 * `Warn` - for potentially harmful events
 * `Error` - for errors
 * `Fatal` - emit fatal error and cause robot to exit(1)

## Bash
```bash
Log "Error" "The robot broke"
```

## PowerShell
```powershell
$bot.Log("Error", "The robot broke")
```

## Python
```python
bot.Log("Error", "The robot broke")
```

## Ruby
```ruby
bot.Log("Error", "The robot broke")
```

# Pause Method

Every language has some means of sleeping / pausing, and this method is provided as a convenience to plugin authors and implemented natively. It takes a single argument, time in seconds.

## Bash
```bash
Say "Be back soon!"
Pause 2
Say "... aaaand I'm back!"
```

## PowerShell
```powershell
$bot.Say("Be back soon!")
$bot.Pause(2)
$bot.Say("... aaaand I'm back!")
```

## Python
```python
bot.Say("Be back soon!")
bot.Pause(2)
bot.Say("... aaaand I'm back!")
```

## Ruby
```ruby
bot.Say("Be back soon!")
bot.Pause(2)
bot.Say("... aaaand I'm back!")
```
