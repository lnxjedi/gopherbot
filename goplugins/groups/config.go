package groups

const groupHelp = `The groups plugin allows you to configure groups, members, and
 group administrators who are able to add and remove members that are
 stored in the robot's memory. For authorization purposes, any user configured
 as a member or administrator, or stored as a member in the robot's long-term
 memory, is considered a member.`

const defaultConfig = `
# For keeping simple shared groups of things
Help:
- Keywords: [ "group", "groups" ]
  Helptext: [ "(bot), help with groups - give general help for using groups"]
- Keywords: [ "group", "groups", "add" ]
  Helptext: [ "(bot), add <user> to the <groupname> group - add a user to a group" ]
- Keywords: [ "group", "groups", "remove" ]
  Helptext: [ "(bot), remove <user> from the <groupname> group - remove a user from a group" ]
- Keywords: [ "group", "groups", "empty" ]
  Helptext: [ "(bot), empty the <groupname> group - remove all dynamic users from a group" ]
- Keywords: [ "list", "group", "groups" ]
  Helptext: [ "(bot), list groups - give a list of all the groups the robot knows about (bot administrators only)" ]
- Keywords: [ "group", "groups", "show", "view" ]
  Helptext: [ "(bot), show the <groupname> group - show the members of a group" ]
CommandMatchers:
- Command: 'help'
  Regex: '(?i:help with groups?)'
- Command: 'add'
  Regex: '(?i:add ([\w-.:]+) to (?:the )?(?:([~\w-'' ]+) )?group)'
  Contexts: [ "user", "group" ]
- Command: 'list'
  Regex: '(?i:group groups)'
- Command: 'remove'
  Regex: '(?i:(?:remove|delete) ([\w-.:]+) from (?:the )?(?:([~\w-'' ]+) )?group)'
  Contexts: [ "user", "group" ]
- Command: 'empty'
  Regex: '(?i:(?:empty|clear) (?:the )?(?:([~\w-'' ]+) )?group)'
  Contexts: [ "group" ]
- Command: 'delete'
  Regex: '(?i:delete (?:the )?(?:([~\w-'' ]+) )?group)'
  Contexts: [ "group" ]
- Command: 'show'
  Regex: '(?i:show (?:the )?(?:([~\w-'' ]+) )?group)'
  Contexts: [ "group" ]
- Command: 'pick'
  Regex: '(?i:(?:pick )(?:an? )?random (?:user )?(?:from )(?:the )?([~\w-'' ]+)?(?: group))'
  Contexts: [ "group" ]
- Command: 'send'
  Regex: '(?i:(?:send me|email) (?:the )?(?:([~\w-'' ]+) )?group)'
  Contexts: [ "group" ]
Config:
  Scope: global # or "channel" if groups aren't shared globally
`
