package memgroups

const memGroupHelp = `The memgroups plugin allows you to configure groups and
 group members who are able to add and remove members from groups that are
 stored in the robot's memory. Configured group members are always considered
 members of the group. If the memgroups plugin is configured with 'Static: true',
 configured members are considered normal members and cannot dynamically add and
 remove members.`

const defaultConfig = `
# For keeping simple shared lists of things
Help:
- Keywords: [ "list", "lists" ]
  Helptext: [ "(bot), help with lists - give general help for using lists"]
- Keywords: [ "list", "lists", "add" ]
  Helptext: [ "(bot), add <item> to the <type> list - add something to a list" ]
- Keywords: [ "list", "lists", "remove" ]
  Helptext: [ "(bot), remove <item> from the <type> list - remove something from a list" ]
- Keywords: [ "list", "lists", "empty" ]
  Helptext: [ "(bot), empty the <type> list - remove all items from a list" ]
- Keywords: [ "list", "lists", "delete" ]
  Helptext: [ "(bot), delete the <type> list - remove the list altogether" ]
- Keywords: [ "list", "lists" ]
  Helptext: [ "(bot), list lists - give a list of all the lists the robot knows about" ]
- Keywords: [ "list", "lists", "email", "send" ]
  Helptext: [ "(bot), send me the <type> list - send a copy of the list by email" ]
- Keywords: [ "list", "lists", "show", "view" ]
  Helptext: [ "(bot), show the <type> list - show the contents of a list" ]
- Keywords: [ "pick", "random", "lists", "list" ]
  Helptext: [ "(bot), pick a random item from the <type> list"]
CommandMatchers:
- Command: 'help'
  Regex: '(?i:help with lists?)'
- Command: 'add'
  Regex: '(?i:add ([-\w .,!?:\/]+) to (?:the )?(?:([~\w-'' ]+) )?list)'
  Contexts: [ "item", "list" ]
- Command: 'list'
  Regex: '(?i:list lists)'
- Command: 'remove'
  Regex: '(?i:(?:remove|delete) ([-\w .,!?:\/]+) from (?:the )?(?:([~\w-'' ]+) )?list)'
  Contexts: [ "item", "list" ]
- Command: 'empty'
  Regex: '(?i:(?:empty|clear) (?:the )?(?:([~\w-'' ]+) )?list)'
  Contexts: [ "list" ]
- Command: 'delete'
  Regex: '(?i:delete (?:the )?(?:([~\w-'' ]+) )?list)'
  Contexts: [ "list" ]
- Command: 'show'
  Regex: '(?i:show (?:the )?(?:([~\w-'' ]+) )?list)'
  Contexts: [ "list" ]
- Command: 'pick'
  Regex: '(?i:(?:pick )(?:an? )?random (?:item )?(?:from )(?:the )?([~\w-'' ]+)?(?: list))'
  Contexts: [ "list" ]
- Command: 'send'
  Regex: '(?i:(?:send me|email) (?:the )?(?:([~\w-'' ]+) )?list)'
  Contexts: [ "list" ]
Config:
  Scope: global # or "channel" if lists aren't shared globally
`
