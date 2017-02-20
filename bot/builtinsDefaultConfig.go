package bot

const helpConfig = `
AllChannels: true
CommandMatchers:
- Command: help
  Regex: '(?i:help ?([\d\w]+)?)'
`

const adminConfig = `
AllChannels: true
RequireAdmin: true
Help:
- Keywords: [ "reload" ]
  Helptext: [ "(bot), reload - have the robot reload configuration files" ]
- Keywords: [ "quit" ]
  Helptext: [ "(bot), quit - request a graceful shutdown" ]
CommandMatchers:
- Command: reload
  Regex: '(?i:reload)'
- Command: quit
  Regex: '(?:quit|exit)'
`

const launchCodesConfig = `
AllChannels: true
Help:
- Keywords: [ "send", "launch", "codes" ]
  Helptext: [ "(bot), send launch codes - one-time send of Google Authenticator string token, for use with TOTP elevation" ]
- Keywords: [ "check", "launch", "codes", "code" ]
  Helptext: [ "(bot), check launch code <code> - verify launch codes operation without launching any missiles" ]
CommandMatchers:
- Command: "send"
  Regex: '(?i:send (?:launch )?codes?)'
- Command: "check"
  Regex: '(?i:check (?:launch )?codes? ([\d]{6}))'
`

const dumpConfig = `
DirectOnly: true
RequireAdmin: true
Help:
- Keywords: [ "dump", "plugin" ]
  Helptext: [ "(bot), dump plugin (default) <plugname> - dump the current or default configuration for the plugin" ]
- Keywords: [ "list", "plugin", "plugins" ]
  Helptext: [ "(bot), list plugins" ]
- Keywords: [ "dump", "robot" ]
  Helptext: [ "(bot), dump (default) robot - dump the current configuration for the robot" ]
CommandMatchers:
- Command: "list"
  Regex: '(?i:list plugins?)'
- Command: "plugdefault"
  Regex: '(?i:dump plugin default ([\d\w]+))'
- Command: "robotdefault"
  Regex: "dump default robot"
- Command: "plugin"
  Regex: '(?i:dump plugin ([\d\w]+))'
- Command: "robot"
  Regex: "dump robot"
`

const logConfig = `
DirectOnly: true
RequireAdmin: true
Help:
- Keywords: [ "log", "logs", "level" ]
  Helptext: [ "(bot), set log level to <trace|debug|info|warning|error> - adjust the logging verbosity" ]
- Keywords: [ "show", "log", "logs" ]
  Helptext: [ "(bot), show log (page X) - display the last or Xth previous page of log output" ]
- Keywords: [ "show", "log", "logs", "level" ]
  Helptext: [ "(bot), show log level - show the current logging level" ]
- Keywords: [ "log", "page", "lines" ]
  Helptext: [ "(bot), set log lines to <number> - set the number of lines returned by show log"]
CommandMatchers:
- Command: "level"
  Regex: '(?i:set log ?level(?: to)? (trace|debug|info|warning|error))'
- Command: "show"
  Regex: '(?i:show logs?(?: page (\d+))?)'
- Command: "showlevel"
  Regex: '(?i:show (?:log ?)?level)'
- Command: "setlines"
  Regex: '(?i:set log ?lines(?: to)? (\d+))'
`
