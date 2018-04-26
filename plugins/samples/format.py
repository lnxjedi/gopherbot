#!/usr/bin/python

import os
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v1 import Robot

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
    print default_config

if command == "format":
    bot = bot.MessageFormat("Variable")
    proto = bot.GetBotAttribute("protocol")
    bot.Say("Hello, %s World!" % proto)
    bot.Say('_italics_ <one> *bold* `code` @parsley')
    bot.Say('_italics_ <one> *bold* `code` @parsley', "raw")
    bot.Say('_italics_ <one> *bold* `code` @parsley', "variable")
    bot.Say('_italics_ <one> *bold* `code` @parsley', "fixed")
    bot.Say('_italics_ <one> *bold* `code` @parsley', "bogus")