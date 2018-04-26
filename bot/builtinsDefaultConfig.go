package bot

const helpConfig = `
AllChannels: true
Help:
- Keywords: [ "info", "information", "robot", "admin", "administrators" ]
  Helptext: [ "(bot), info | tell me about yourself - provide useful information for admins, or a list of admins" ]
CommandMatchers:
- Command: help
  Regex: '(?i:help ?([\d\w]+)?)'
- Command: info
  Regex: '(?i:info|tell me about yourself|about|information)'
`

const adminConfig = `
AllChannels: true
AllowDirect: true
RequireAdmin: true
Help:
- Keywords: [ "reload" ]
  Helptext: [ "(bot), reload - have the robot reload configuration files" ]
- Keywords: [ "quit" ]
  Helptext: [ "(bot), quit - request a graceful shutdown, waiting for all plugins to finish" ]
- Keywords: [ "abort" ]
  Helptext: [ "(bot), abort - request an immediate shutdown without waiting for plugins to finish" ]
- Keywords: [ "debug" ]
  Helptext: [ "(bot), debug plugin <pluginname> (verbose) - turn on debugging for the named plugin, optionally verbose" ]
- Keywords: [ "debug" ]
  Helptext: [ "(bot), stop debugging - turn off debugging" ]
CommandMatchers:
- Command: reload
  Regex: '(?i:reload)'
- Command: quit
  Regex: '(?i:quit|exit)'
- Command: abort
  Regex: '(?i:abort)'
- Command: "debug"
  Regex: '(?i:debug (?:plugin )?([\d\w-.]+)(?: (verbose))?)'
- Command: "stop"
  Regex: '(?i:stop debugging)'
`

const dumpConfig = `
DirectOnly: true
RequireAdmin: true
Help:
- Keywords: [ "dump", "plugin" ]
  Helptext: [ "(bot), dump plugin (default) <plugname> - dump the current or default configuration for the plugin" ]
- Keywords: [ "list", "plugin", "plugins" ]
  Helptext: [ "(bot), list (disabled) plugins - list all known plugins, or list disabled plugins with the reason disabled" ]
- Keywords: [ "dump", "robot" ]
  Helptext: [ "(bot), dump robot - dump the current configuration for the robot" ]
CommandMatchers:
- Command: "list"
  Regex: '(?i:list( disabled)? plugins?)'
- Command: "plugdefault"
  Regex: '(?i:dump plugin default ([\d\w-.]+))'
- Command: "plugin"
  Regex: '(?i:dump plugin ([\d\w-.]+))'
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
  Regex: '(?i:set log ?level(?: to)? (trace|debug|info|warn|error))'
- Command: "show"
  Regex: '(?i:show logs?(?: page (\d+))?)'
- Command: "showlevel"
  Regex: '(?i:show (?:log ?)?level)'
- Command: "setlines"
  Regex: '(?i:set log ?lines(?: to)? (\d+))'
`
