package timetrack

const defaultConfig = `
Help:
- Keywords: [ "track", "time" ]
  Helptext: [ "(bot), track time for <task> - start tracking time spent on a task" ]
- Keywords: [ "track", "time", "checkin", "check" ]
  Helptext: [ "(bot), check in - uptime time for tasks being tracked"]
CommandMatchers:
- Command: track
  Regex: '(?i:track(?:time for )?([-\w ,]))'
- Command: checkin
  Regex: '(?i:check ?in)'
Config:
  AllowDoubleCharge: false # whether time can be allocated to multiple tasks
  ReportingPeriod: month   # or week
  WeekStart: Sunday
  CheckinMinutes: 20       # how often to poll the user
  CheckinContext: channel  # or 'direct' for robot to request checkin via DM
  AutoClose: 120           # minutes to wait before closing a task
  AutoCloseCharge: 10      # how many minutes to charge to an autoclosed task`
