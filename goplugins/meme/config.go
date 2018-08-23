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
- Keywords: [ "meme", "farnsworth", "news" ]
  Helptext:
  - "(bot), Good news everyone (<something>) - let Professor Farnsworth deliver the good news"
  - "(bot), farnsworth <something>/<something> - Professor Farnsworth expounds"
- Keywords: [ "meme", "roy", "phone" ]
  Helptext: [ "(bot), roy phone <something>(/<something>) - Roy provides phone support" ]
- Keywords: [ "meme", "gosh" ]
  Helptext: [ "(bot), <something>, gosh! - Let Napoleon Dynamite express your indignation" ]
- Keywords: [ "meme", "best", "worst" ]
  Helptext: [ "(bot), this is pretty much the best/worst <something> - Napoleon expresses his opinion" ]
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
- Keywords: [ "meme", "matrix" ]
  Helptext: [ "(bot), What if I told you <something> - let Morpheus blow their minds" ]
- Keywords: [ "meme", "matrix" ]
  Helptext: [ "(bot), morpheus <something>/<something>" ]

CommandMatchers:
- Command: "1509839"
  Regex: '(?i:picard omg ([^/]+)(?:/([^/]+))?)'
- Command: "245898"
  Regex: '(?i:picard wt[hf] ([^/]+)(?:/([^/]+))?)'
- Command: "29106534"
  Regex: '(?i:roy phone ([^/]+)(?:/([^/]+))?)'
- Command: "7163250"
  Regex: '(?i:((?:good|great) news,? (?:everyone|everybody)),? (.+))'
- Command: "7163250"
  Regex: '(?i:farnsworth ([^/]+)(?:/([^/]+))?)'
- Command: "18304105"
  Regex: '(?i:(.+,?) (gosh!?))'
- Command: "8070362"
  Regex: '(?i:(.+ pretty much the) ((?:best|worst) .+))'
- Command: "20509936"
  Regex: '(?i:(.+ skills?) ((?:with|in) .+))'
- Command: "61579"
  Regex: '(?i:(one does not simply) (.+))'
- Command: "47779539"
  Regex: '(?i:(you .+) (prepare to die!?))'
- Command: "61546"
  Regex: '(?i:(brace yourselves,?) (.+))'
- Command: "61527"
  Regex: '(?i:(y u no) (.+))'
- Command: "33301480"
  Regex: '(?i:(what if I told you) (.+))'
- Command: "33301480"
  Regex: '(?i:morpheus ([^/]+)(?:/([^/]+))?)'
# Custom configuration for memes - you need to supply a username and password,
# and a map of commands to meme ID #.
Config:
  Username: '<your-imgflip-username>'
  Password: '<your-password>'
`
