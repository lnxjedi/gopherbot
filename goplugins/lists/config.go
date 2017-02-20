package lists

const defaultConfig = `
# For keeping simple shared lists of things
Help:
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
CommandMatchers:
- Command: 'add'
  Regex: '(?i:add ([\w\d- ]+) to the ([\w]+) list)'
- Command: 'list'
  Regex: '(?i:list lists)'
- Command: 'remove'
  Regex: '(?i:remove ([\w\d- ]+) from the ([\w]+) list)'
- Command: 'empty'
  Regex: '(?i:empty the ([\w]+) list)'
- Command: 'delete'
  Regex: '(?i:delete the ([\w]+) list)'
- Command: 'show'
  Regex: '(?i:show the ([\w]+) list)'
- Command: 'send'
  Regex: '(?i:(?:send me|email) the ([\w]+) list)'
`
