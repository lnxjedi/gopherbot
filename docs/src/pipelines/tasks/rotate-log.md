___

**rotate-log** - *privileged*

Usage: `AddTask rotate-log <extension>`

Useful for e.g. nightly log rotation jobs, when the robot has been started with e.g.: `gopherbot -d -l robot.log`. For example:
```
AddTask rotate-log "log.$(date +%a)"
```

This would keep daily logs of the form `robot.log.Mon`, etc.