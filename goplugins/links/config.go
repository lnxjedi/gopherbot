package links

const defaultConfig = `
Help:
- Keywords: [ "link", "links", "add" ]
  Helptext: [ "(bot), link <word/phrase> to <http://...> - save a link with a single word/phrase key" ]
- Keywords: [ "link", "links", "save", "add" ]
  Helptext: [ "(bot), save link <http://...> - save a link and prompt for multiple word/phrase keys"]
- Keywords: [ "link", "links", "find", "lookup", "search" ]
  Helptext:
  - "(bot), (find|lookup) <keyword/phrase> - find links with keys containing a keyword or phrase"
  - "(bot), look <keyword/phrase> up"
- Keywords: [ "link", "links", "remove" ]
  Helptext: [ "(bot), remove <http://...> - remove a link" ]
- Keywords: [ "link", "links" ]
  Helptext: [ "(bot), help with links - give a description of the links plugin" ]
- Keywords: [ "link", "links", "list", "show" ]
  Helptext: [ "(bot), (list|show) links - list all the links the robot knows" ]
CommandMatchers:
- Command: 'help'
  Regex: '(?i:help with links?)'
- Command: 'add'
  Regex: '(?i:link ([-\w ,''!."+=?&@#()/]+) to ((?:http(?:s)?:\/\/)?(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b(?:[-a-zA-Z0-9@:%_\+.~#?&//=]*)))'
  Contexts: [ "item", "link" ]
- Command: 'add'
  Regex: '(?i:link ([-\w ,''!."+=?&@#()/]+) to (it))'
  Contexts: [ "item", "link" ]
- Command: 'save'
  Regex: '(?i:save (?:link )?((?:http(?:s)?:\/\/)?(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b(?:[-a-zA-Z0-9@:%_\+.~#?&//=]*)))'
  Contexts: [ "link" ]
- Command: 'remove'
  Regex: '(?i:(?:remove|delete) (?:link )?((?:http(?:s)?:\/\/)?(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b(?:[-a-zA-Z0-9@:%_\+.~#?&//=]*)))'
  Contexts: [ "link" ]
- Command: 'find'
  Regex: '(?i:(?:find|look ?up) ([-\w ,''!."+=?&@#()/]+))'
  Contexts: [ "item" ]
- Command: 'list'
  Regex: '(?i:(?:show|list) links)'
- Command: 'find'
  Regex: '(?i:look ([-\w ,''!."+=?&@#()/]+) up)'
  Contexts: [ "item" ]
ReplyMatchers:
- Regex: '([-\w ,''!."+=?&@#()/]+)'
  Label: "lookup"
Config:
  Scope: global # or "channel"
`

const longHelp = `The links plugin stores URLs and associates them with a text key that can
be words or phrases. The 'link' command stores a link and key in one command, and the
'save' command will prompt the user to enter the key. The lookup command
will return all links whose key contains the provided word or phrase,
case insensitive. Links can be deleted with the 'remove' command.`
