package memes

const defaultConfig = `
# You can delete everything up to Config: below
Disabled: true
Help:
- Keywords: [ "meme", "gosh" ]
  Helptext: [ "(bot), <something>, gosh! - Let Napoleon Dynamite express your indignation" ]
- Keywords: [ "meme", "best", "worst" ]
  Helptext: [ "(bot), this is pretty much the best/worst <something> ever <something> - Napoleon expresses his opinion" ]
- Keywords: [ "meme", "skill", "skills" ]
  Helptext: [ "(bot), <something> skill(s) <something> - Hear about Napoleon's incredible skills" ]
- Keywords: [ "meme", "simply" ]
  Helptext: [ "(bot), one does not simply <do something> - Summon Boromir to make your point" ]
- Keywords: [ "meme", "prepare" ]
  Helptext: [ "(bot), you <did something>, prepare to die - Let Inigo threaten your friends" ]

CommandMatchers:
- Command: gosh
  Regex: '(?mi:([\w\n’'' ]+), gosh!)'
- Command: prettymuch
  Regex: '(?i:([\w''’ ]+) pretty much the ((?:best|worst) [\w''’]+) ever ([\w''’!]+))'
- Command: skills
  Regex: '(?mi:([\w\n''’ ]+) (skills?) ([\w\n''’! ]+))'
- Command: simply
  Regex: '(?mi:one does not simply ([\w!\n''’ ]+))'
- Command: prepare
  Regex: '(?mi:you ([\w!\n''’ ]+),? prepare to die)'
# Custom configuration for memes - you need at least this section with
# username and password supplied.
Config:
  Username: '<your-imgflip-username>'
  Password: '<your-password>'
`
