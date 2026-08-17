# Required Bot Info

Earlier versions of **Gopherbot** expected to get info about the robot - name, handle, etc. - from the protocol connector; however, that can't be counted on. For your robot to respond to a command like `floyd, ping`, it will need a minimal `BotInfo` section in `robot.yaml`:

```yaml
BotInfo:
  UserName: floyd
```

To provide other information about the robot to the `GetBotAttribute()` method, you might want to fill out the entire structure:

```yaml
BotInfo:
  UserName: floyd
  Email: floyd@linuxjedi.org
  FullName: Floyd Gopherbot
  FirstName: Floyd
  LastName: Gopherbot
```

The robot's single-character `Alias` is still specified by itself, as before; e.g. `Alias: ";"`.
