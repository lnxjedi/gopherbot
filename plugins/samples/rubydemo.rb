#!/usr/bin/ruby

# boilerplate
require ENV["GOPHER_INSTALLDIR"] + '/lib/gopherbot_v1'

bot = Robot.new()
# /boilerplate

=begin
The defaultConfig is a multi-line YAML formatted document that specifies
this plugin's default configuration for Gopherbot. It may include any of the
fields in https://godoc.org/github.com/lnxjedi/gopherbot/bot#Plugin, as
well as arbitrary YAML for config data that a bot admin might want to
override.
=end
defaultConfig = <<'DEFCONFIG'
---
Help:
- Keywords: [ "ruby" ]
  Helptext: [ "(bot), ruby (me!) - prove that ruby plugins work" ]
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
- Regex: (?i:ruby( me)?!?)
  Command: ruby
- Regex: (?i:listen( to me)?!?)
  Command: listen
- Regex: '(?i:remember(?: (slowly))? ([-\w .,!?:\/]+))'
  Command: remember
  Contexts: [ "", "item" ]
- Regex: (?i:recall ?([\d]+)?)
  Command: recall
- Regex: (?i:forget ([\d]{1,2}))
  Command: forget
- Regex: (?i:check me)
  Command: check
Config:
  Replies:
  - "Consider yourself rubied"
  - "Waaaaaait a second... what do you mean by that?"
  - "I'll ruby you, but not right now - I'll wait 'til you're least expecting it..."
  - "Crap, sorry - all out of rubies"
DEFCONFIG

command = ARGV.shift()

case command
when "configure"
  puts defaultConfig
  exit
when "ruby"
  bot.Say("Sure, #{bot.GetSenderAttribute("firstName")}!")
  sleep 1.5
  bot.Say(bot.RandomString(bot.GetTaskConfig()["Replies"]))
when "listen"
  dbot = bot.Direct()
  rep = dbot.PromptForReply("SimpleString", "Ok, what do you want to tell me?")
  if rep.ret == Robot::Ok
    dbot.Say("I hear what you're saying - '#{rep}'")
  else
    bot.Say("I'm sorry, I'm not sure what you're trying to tell me - did you put funny characters in your reply?")
  end
when "remember"
  speed = ARGV[0]
  thing = ARGV[1]
  memory = bot.CheckoutDatum("memory", true)
  remembered = false
  if memory.exists
    if memory.datum.include?(thing)
      bot.Say("That's already one of my fondest memories")
      bot.CheckinDatum(memory)
    else
      remembered =true
      memory.datum.push(thing)
    end
  else
    remembered = true
    memory.datum = [ thing ]
  end
  if remembered
    if speed == "slowly"
      bot.Say("Ok, I'll remember \"#{thing}\" ... but sloooowly")
    else
      bot.Say("Ok, I'll remember \"#{thing}\"")
    end
    if speed == "slowly"
      bot.Pause(4)
    end
    ret = bot.UpdateDatum(memory)
    if speed != "slowly" && ret == Robot::Ok
      bot.Say("committed to memory")
    end
    if ret != Robot::Ok && speed != "slowly"
      bot.Say("Dang it, having problems with my memory")
    end
  end
when "recall"
  memory = bot.CheckoutDatum("memory", false)
  if memory.exists
    if ARGV[0].length > 0
      mnum = ARGV[0].to_i - 1
      if mnum < 0
        bot.Say("I can't make out what you want me to remember")
      elsif mnum >= memory.datum.length()
        bot.Say("I don't remember that many things!")
      else
        bot.CheckinDatum(memory)
        bot.Say(memory.datum[mnum])
      end
    else
      reply = "Here's what I remember:\n"
      memory.datum.each_index { |i|
        index = i + 1
        reply += index.to_s + ": " + memory.datum[i] + "\n"
      }
      bot.CheckinDatum(memory)
      bot.Say(reply)
    end
  else
    bot.Say("Sorry - I don't remember anything!")
  end
when "forget"
  i = ARGV[0].to_i - 1
  memory = bot.CheckoutDatum("memory", true)
  memories = memory.datum
  if i >= 0 && memories.class == Array && memories[i]
    bot.Say("Ok, I'll forget \"#{memories[i]}\"")
    memories.delete_at(i)
    bot.UpdateDatum(memory)
  else
    bot.CheckinDatum(memory)
    bot.Say("Gosh, I guess I never remembered that in the first place!")
  end
when "check"
  isAdmin = bot.CheckAdmin()
  if isAdmin
    bot.Say("Ok, it looks like you're an administrator")
  else
    bot.Say("Well, you're not an administrator")
  end
  bot.Pause(1)
  bot.Say("Now I'll request elevation...")
  success = bot.Elevate(true)
  if success
    bot.Say("Everything looks good, mac!")
  else
    bot.Say("You failed to elevate, homie, I'm calling the cops!")
  end
  bot.Log("info", "Checked out #{bot.user}, admin: #{isAdmin.to_s}, elavate check: #{success.to_s}")
end
