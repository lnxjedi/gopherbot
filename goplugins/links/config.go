package links

const defaultConfig = `
Help:
- Keywords: [ "link", "links", "add" ]
  Helptext: [ "(bot), link <word/phrase> to <http://...> - save a link for the given word/phrase only" ]
- Keywords: [ "link", "links", "save", "add" ]
  Helptext: [ "(bot), save link <http://...> - save a link and prompt for keywords / phrases"]
- Keywords: [ "link", "links", "find", "lookup" ]
  Helptext: [ "(bot), (find|lookup) <keyword/phrase> - look up links for the given keyword/phrase" ]
- Keywords: [ "link", "links", "remove" ]
  Helptext: [ "(bot), remove <http://...> - remove a link" ]
- Keywords: [ "link", "links", "search", "find" ]
  Helptext: [ "(bot), search links for <word>"]
CommandMatchers:
#- Command: 'help'
#  Regex: '(?i:help with links?)'
- Command: 'add'
  Regex: '(?i:link ([-\w \'']+) to ((?:http(?:s)?:\/\/)?(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b(?:[-a-zA-Z0-9@:%_\+.~#?&//=]*)))'
- Command: 'save'
  Regex: '(?i:save (?:link )?((?:http(?:s)?:\/\/)?(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b(?:[-a-zA-Z0-9@:%_\+.~#?&//=]*)))'
- Command: 'remove'
  Regex: '(?i:(?:remove|delete) (?:link )?((?:http(?:s)?:\/\/)?(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b(?:[-a-zA-Z0-9@:%_\+.~#?&//=]*)))'
- Command: 'remove'
  Regex: '(?:(?:remove|delete) (it))'
  Contexts: [ "link" ]
- Command: 'find'
  Regex: '(?i:(?:find|look ?up) ([-\w \'']+))'
- Command: 'search'
  Regex: '(?i:search links for ([-\w\'']+))'
ReplyMatchers:
- Regex: '([-\w ,\'']+)'
  Label: "lookup"
Config:
  Scope: global # or "channel"
`
