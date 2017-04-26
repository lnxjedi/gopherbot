package links

const defaultConfig = `
Help:
- Keywords: [ "link", "links", "add" ]
  Helptext: [ "(bot), link <words> to <http://...> - save a link for the given words" ]
- Keywords: [ "link", "links", "find" ]
  Helptext: [ "(bot), find <keyword> - look up links for the given keyword" ]
- Keywords: [ "link", "links", "remove" ]
  Helptext: [ "(bot), remove <http://...> - remove a link" ]
CommandMatchers:
- Command: 'help'
  Regex: '(?i:help with links?)'
- Command: 'add'
  Regex: '(?i:add ([-\w .,!?:\/]+) to (?:the )?(?:([\w-_ ]+) )?link)'
  Contexts: [ "item", "link" ]
- Command: 'link'
  Regex: '(?i:link links)'
- Command: 'remove'
  Regex: '(?i:(?:remove|delete) ([-\w .,!?:\/]+) from (?:the )?(?:([\w-_ ]+) )?link)'
  Contexts: [ "item", "link" ]
- Command: 'empty'
  Regex: '(?i:(?:empty|clear) (?:the )?(?:([\w-_ ]+) )?link)'
  Contexts: [ "link" ]
- Command: 'delete'
  Regex: '(?i:delete (?:the )?(?:([\w-_ ]+) )?link)'
  Contexts: [ "link" ]
- Command: 'show'
  Regex: '(?i:show (?:the )?(?:([\w-_ ]+) )?link)'
  Contexts: [ "link" ]
- Command: 'pick'
  Regex: '(?i:(?:pick )?(?:an? )?random (?:item )?(?:from )?(?:the )?(?:([\w-_ ]+) )?(?:link)?)'
  Contexts: [ "link" ]
- Command: 'send'
  Regex: '(?i:(?:send me|email) (?:the )?(?:([\w-_ ]+) )?link)'
  Contexts: [ "link" ]
`
