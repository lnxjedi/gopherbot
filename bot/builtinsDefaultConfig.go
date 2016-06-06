package bot

const helpConfig = `
AllChannels: true
CommandMatches:
- Command: help,
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
CommandMatches:
- Command: reload
  Regex: '(?i:reload)'
- Command: quit
  Regex: '(?:quit|exit)'
`

const launchCodesConfig = `
AllChannels: true
Help:
- Keywords: [ "send", "launch", "codes" ]
  Helptext: [ "(bot), send launch codes - one-time send of URL for configuring Google Authenticator, for 2FA commands" ]
- Keywords: [ "check", "launch", "codes", "code" ]
  Helptext: [ "(bot), check launch code <code> - verify launch codes operation without launching any missiles" ]
CommandMatches:
- Command: "send"
  Regex: '(?i:send (?:launch )?codes?)'
- Command: "check"
  Regex: '(?i:check (?:launch )?codes? ([\d]{6}))'
`
