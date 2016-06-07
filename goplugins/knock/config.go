package knock

const defaultConfig = `
Help:
- Keywords: [ "knock", "joke" ]
  Helptext: [ "(bot), tell me a (knock-knock) joke" ]
CommandMatches:
- Command: knock
  Regex: '(?i:tell me a(?:nother)?(?: knock[- ]knock)? joke)'
ReplyMatchers:
- Command: whosthere
  Regex: '(?i:who.?s there\??)'
- Command: who
  Regex: '(?i:[\w ]+ who\??)'
Config:
  Jokes:
  - { First: "To", Second: "\"To *whom*\"" }
  - { First: "Doctor", Second: "Man, I wish!" }
  - { First: "Eskimo", Second: "Eskimo questions, I tell no lies" }
  - { First: "Spell", Second: "W H O" }
  - { First: "Toby", Second: "Toby or not Toby - that is the question" }
  - { First: "Ya", Second: "Well, I'm happy to see you, too!" }
  - { First: "Keith", Second: "Keith me, my thweet preenthith!" }
  - { First: "Police", Second: "Police let me in, it's cold out here!" }
  - { First: "Isabel", Second: "Isabel working, or should I keep knocking?" }
  - { First: "Etch", Second: "Bless you!" }
  - { First: "Claire", Second: "Claire the way, I'm coming in!" }
  - { First: "Radio", Second: "Radio not, here I come!" }
  - { First: "Howard", Second: "Howard I know?" }
  - { First: "Cereal", Second: "Cereal pleasure to meet you!" }
  - { First: "Alpaca", Second: "Alpaca the suitcase if you'll loada the car!" }
  - { First: "Wooden shoe", Second: "Wooden shoe love to hear another knock-knock joke?" }
  - { First: "Nana", Second: "Nana yer business, open up!" }
  - { First: "Harry", Second: "Harry up and let me in!" }
  - { First: "Adolph", Second: "Adolph ball hit me in the mouf - dat why I talk dis way!" }
  - { First: "Omar", Second: "Omar goodness, wrong house!" }
  - { First: "Panther", Second: "Panther no panth, I’m going thwimming!" }
  - { First: "Oswald", Second: "Oswald my bubble gum!" }
  - { First: "Olive", Second: "Olive you and I don't care who knows it!" }
  - { First: "Cargo", Second: "No, car go \"beep beep\"" }
  - { First: "Goliath", Second: "Goliath down, thou looketh tired" }
  - { First: "Wendy", Second: "Wendy bell works again, I'll stop knocking" }
  - { First: "Figs", Second: "Figs your dang doorbell already!" }
  - { First: "Moustache", Second: "Moustache you a question, but I’ll shave it for later!" }
  - { First: "Broken pencil", Second: "Ah, forget it - it's pointless" }
  - { First: "Tank", Second: "You're welcome!" }
  - { First: "Al", Second: "Al give you a kiss if you'll open the door!" }
  - { First: "Weevil", Second: "Weevil weevil rock you" }
  - { First: "Frank", Second: "Frank you for being my friend!" }
  - { First: "Dishes", Second: "Dishes a nice place ya got here" }
  Openings:
  - "Ok, I know a good one!"
  - "Hrm... ok, this is one of my favorites..."
  - "I'll see if I can think of one..."
  - "Another robot told me this one, tell me if you think it's funny"
  - "I found lame joke on the Internet - but it's kinda funny when a robot tells it"
  - "I'll ask Watson(tm) if he knows any good ones and get back to you in a jiffy..."
  - "Hang on while I Google that for you (just kidding ;-)"
  - "Sure - Siri told me this one, but I think it's kind of dumb"
  - "Ok, here's a funny one I found in Hillary's email..."
  - "Yeah! I LOVE telling jokes!"
  - "Alright - I'll see if I can make my voice sound funny"
  Phooey:
  - "Ah, you're no fun"
  - "What, don't you like a good knock-knock joke?"
  - "Ok, maybe another time"
`
