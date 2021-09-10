#!/usr/bin/python3

import sys
from gopherbot_v2 import Robot

bot = Robot()

default_config = '''
Channels:
- random
AllowDirect: false
Help:
- Keywords: [ "format", "world" ]
  Helptext: [ "(bot), format world - exercise formatting options" ]
CommandMatchers:
- Regex: '(?i:format world)'
  Command: "format"
'''

executable = sys.argv.pop(0)
command = sys.argv.pop(0)

if command == "configure":
    print(default_config)

if command == "format":
    bot = bot.MessageFormat("Variable")
    proto = bot.GetBotAttribute("protocol")
    bot.Say("Hello, %s World!" % proto)
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley')
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley', "raw")
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley', "variable")
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley', "fixed")
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley', "bogus")
