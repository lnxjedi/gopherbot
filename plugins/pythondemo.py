#!/usr/bin/python

import os
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v1 import Robot

bot = Robot()

default_config = '''
---
Help:
- Keywords: [ "bashecho", "bash", "echo" ]
  Helptext: [ "(bot), bashecho <simple text> - call the bash echo plugin to echo a phrase" ]
- Keywords: [ "python" ]
  Helptext: [ "(bot), python (me!) - prove that python plugins work" ]
- Keywords: [ "listen" ]
  Helptext: [ "(bot), listen (to me!) - ask a question" ]
- Keywords: [ "remember", "memory" ]
  Helptext: [ "(bot), remember <anything> - prove the robot has a brain(tm)" ]
- Keywords: [ "recall", "memory" ]
  Helptext: [ "(bot), recall - prove the robot has a brain(tm)" ]
- Keywords: [ "forget", "memory" ]
  Helptext: [ "(bot), forget <#> - ask the robot to forget one of it's remembered 'facts'" ]
- Keywords: [ "check" ]
  Helptext: [ "(bot), check me - get the bot to check you out" ]
CommandMatchers:
- Regex: '(?i:bashecho ([.;!\d\w-, ]+))'
  Command: bashecho
- Regex: (?i:python( me)?!?)
  Command: python
- Regex: (?i:listen( to me)?!?)
  Command: listen
- Regex: (?i:remember ([-\w .,!?:\/]+))
  Command: remember
  Contexts: [ "item" ]
- Regex: (?i:(?:recall|memories))
  Command: recall
- Regex: '(?i:forget #?([\d]{1,2}))'
  Command: forget
- Regex: (?i:check me)
  Command: check
Config:
  Replies:
  - "You has the pythons"
  - "Are you JOKING?!? Pythons are DANGEROUS!!"
  - "Eh, how about a grass snake?"
  - "Sorry, I left it in my other jacket"
'''

executable = sys.argv.pop(0)
command = sys.argv.pop(0)

if command == "configure":
    print default_config

if command == "python":
    bot.Say("Sure, %s, gimme a sec to look for it..." % bot.GetSenderAttribute("firstName"))
    bot.Pause(1.5)
    bot.Say(bot.RandomString(bot.GetPluginConfig()["Replies"]))

if command == "bashecho":
    status = bot.CallPlugin("echo", "echo", sys.argv.pop(0))
    if status != Robot.Normal:
        bot.Say("Uh-oh, there was a problem calling the echo plugin, status: %d" % status)

if command == "listen":
    dbot = bot.Direct()
    rep = dbot.PromptForReply("SimpleString", "Ok, what do you want to tell me?")
    if rep.ret == Robot.Ok:
        dbot.Say("I hear what you're saying - \"%s\"" % rep)
    else:
        bot.Say("I'm sorry, I had a hard time hearing your replay - funny characters? Take too long?")

if command == "remember":
    thing = sys.argv.pop(0)
    bot.Say("Ok, I'll remember \"%s\"" % thing)
    memory = bot.CheckoutDatum("memory", True)
    if memory.exists:
        memory.datum.append(thing)
    else:
        memory.datum = [ thing ]
    ret = bot.UpdateDatum(memory)
    if ret != Robot.Ok:
        bot.Say("Uh-oh, I must be gettin' old - having memory problems!")

if command == "recall":
    memory = bot.CheckoutDatum("memory", False)
    if memory.exists:
        reply = [ "Here's everything I can remember:" ]
        for i in range(0, len(memory.datum)):
            reply.append("#%d: %s" % ( i + 1, memory.datum[i] ))
        bot.Say("\n".join(reply))
    else:
        bot.Say("Gee, I don't remember ANYTHING")

if command == "forget":
    item = int(sys.argv.pop(0)) - 1
    if item >= 0:
        memory = bot.CheckoutDatum("memory", True)
        if memory.exists and len(memory.datum) > item:
            bot.Say("Ok, I'll forget \"%s\"" % memory.datum[item])
            memory.datum = memory.datum[:item] + memory.datum[item+1:]
            bot.UpdateDatum(memory)
        else:
            bot.CheckinDatum(memory)
            bot.Say("I don't see that item number in my memories")
    else:
        bot.Say("A wise guy, eh?")

if command == "check":
    if bot.CheckAdmin():
        bot.Say("Ah - you're an administrator!")
    else:
        bot.Say("Huh, looks like you're a regular schmoe")
    bot.Pause(1)
    bot.Say("Ok, let's see if you're able to elevate...")
    if bot.Elevate(True):
        bot.Say("Great, I should be able to do some real work for you")
    else:
        bot.Say("Uh-oh - forget getting me to do any real work for you!")
