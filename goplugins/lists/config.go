package lists

const listHelp = `The list plugin allows you to manage simple lists of items,
 such as a todo list, lunch spots, etc. Lists default to global scope, but can be
 configured with per-channel scope. Note that 'lists' is a context aware plugin, and
 will remember the list or item being discussed; so e.g. you can follow 'add milk
 to the grocery list' with 'add hamburgers to the list' and the robot will
 know you mean the grocery list; also, 'add it to the dinner list' would add
 hamburgers to the dinner list. List names are always lowercased, but items are
 stored with case preserved and compared in a case-insensitive manner.`

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
  Regex: '(?i:add ([-\w .,!?:\/''’"]+) to (?:the )?(?:([-\w .,!?:\/''’"]+) )?list)'
  Contexts: [ "item:it", "list" ]
- Command: 'list'
  Regex: '(?i:list lists)'
- Command: 'remove'
  Regex: '(?i:(?:remove|delete) ([-\w .,!?:\/''’"]+) from (?:the )?(?:([-\w .,!?:\/''’"]+) )?list)'
  Contexts: [ "item:it", "list" ]
- Command: 'empty'
  Regex: '(?i:(?:empty|clear) (?:the )?(?:([-\w .,!?:\/''’"]+) )?list)'
  Contexts: [ "list" ]
- Command: 'delete'
  Regex: '(?i:delete (?:the )?(?:([-\w .,!?:\/''’"]+) )?list)'
  Contexts: [ "list" ]
- Command: 'show'
  Regex: '(?i:show (?:the )?(?:([-\w .,!?:\/''’"]+) )?list)'
  Contexts: [ "list" ]
- Command: 'pick'
  Regex: '(?i:(?:pick )(?:an? )?random (?:item )?(?:from )(?:the )?([-\w .,!?:\/''’"]+)?(?: list))'
  Contexts: [ "list" ]
- Command: 'send'
  Regex: '(?i:(?:send me|email) (?:the )?(?:([-\w .,!?:\/''’"]+) )?list)'
  Contexts: [ "list" ]
Config:
  Scope: global # or "channel" if lists aren't shared globally
`
