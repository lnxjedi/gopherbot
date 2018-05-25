package memes

const defaultConfig = `
# You can delete everything up to Config: below
# To edit the memes or add your own, copy all of the help and command matchers
# to your own local config. See: https://api.imgflip.com/
Disabled: true
Help:
- Keywords: [ "meme", "picard", "omg" ]
  Helptext: [ "(bot), picard omg <something>(/<something>) - Picard facepalm" ]
- Keywords: [ "meme", "picard", "wth", "wtf" ]
  Helptext: [ "(bot), picard wth <something>(/<something>) - Picard WTH" ]
- Keywords: [ "meme", "gosh" ]
  Helptext: [ "(bot), <something>, gosh! - Let Napoleon Dynamite express your indignation" ]
- Keywords: [ "meme", "best", "worst" ]
  Helptext: [ "(bot), this is pretty much the best/worst <something> ever <something> - Napoleon expresses his opinion" ]
- Keywords: [ "meme", "skill", "skills" ]
  Helptext: [ "(bot), <something> skill(s) with <something> - Hear about Napoleon's incredible skills" ]
- Keywords: [ "meme", "simply" ]
  Helptext: [ "(bot), one does not simply <do something> - Summon Boromir to make your point" ]
- Keywords: [ "meme", "prepare" ]
  Helptext: [ "(bot), you <did something>, prepare to die - Let Inigo threaten your friends" ]
- Keywords: [ "meme", "brace" ]
  Helptext: [ "(bot), brace yourselves, <something> - Boromir warns your" ]
- Keywords: [ "meme" ]
  Helptext: [ "(bot), Y U no <something> - express your angst" ]

CommandMatchers:
- Command: "1509839"
  Regex: '(?i:picard omg ([^/]+)(?:/([^/]+))?)'
- Command: "245898"
  Regex: '(?i:picard wt[hf] ([^/]+)(?:/([^/]+))?)'
- Command: "18304105"
  Regex: '(?i:([\w’'' ]+,) (gosh!))'
- Command: "8070362"
  Regex: '(?i:([\w''’ ]+ pretty much the) ((?:best|worst) [\w''’]+ ever [\w''’!]+))'
- Command: "20509936"
  Regex: '(?i:([\w''’ ]+ skills?) ((with|in) [\w''’! ]+))'
- Command: "61579"
  Regex: '(?i:(one does not simply) ([\w!\n''’ ]+))'
- Command: "47779539"
  Regex: '(?i:(you [\w!''’ ]+,?) (prepare to die!?))'
- Command: "61546"
  Regex: '(?i:(brace yourselves,?) ([\w''’ !]+))'
- Command: "61527"
  Regex: '(?i:(y u no) ([\w''’ !?]+))'
# Custom configuration for memes - you need to supply a username and password,
# and a map of commands to meme ID #.
Config:
  Username: '<your-imgflip-username>'
  Password: '<your-password>'
`
