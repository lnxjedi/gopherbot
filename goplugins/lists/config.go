package lists

const listHelp = `The list plugin allows you to manage simple lists of items,
such as a TODO list, lunch spots, etc. List items can be no longer than 42
characters, and list names no longer than 21. When referring to list names,
spaces will be replaced with dashes, so "meeting items" and "meeting-items" both
refer to the "meeting-items" list. Note that lists is context aware, and will
remember the list or item being discussed.`

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
  Regex: '(?i:add ([-\w .,!?:\/]+) to (?:the )?(?:([\w-_ ]+) )?list)'
  Contexts: [ "item", "list" ]
- Command: 'list'
  Regex: '(?i:list lists)'
- Command: 'remove'
  Regex: '(?i:(?:remove|delete) ([-\w .,!?:\/]+) from (?:the )?(?:([\w-_ ]+) )?list)'
  Contexts: [ "item", "list" ]
- Command: 'empty'
  Regex: '(?i:(?:empty|clear) (?:the )?(?:([\w-_ ]+) )?list)'
  Contexts: [ "list" ]
- Command: 'delete'
  Regex: '(?i:delete (?:the )?(?:([\w-_ ]+) )?list)'
  Contexts: [ "list" ]
- Command: 'show'
  Regex: '(?i:show (?:the )?(?:([\w-_ ]+) )?list)'
  Contexts: [ "list" ]
- Command: 'pick'
  Regex: '(?i:(?:pick )?(?:an? )?random (?:item )?(?:from )?(?:the )?(?:([\w-_ ]+) )?(?:list)?)'
  Contexts: [ "list" ]
- Command: 'send'
  Regex: '(?i:(?:send me|email) (?:the )?(?:([\w-_ ]+) )?list)'
  Contexts: [ "list" ]
`
