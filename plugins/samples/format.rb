#!/usr/bin/ruby

# boilerplate
require ENV["GOPHER_INSTALLDIR"] + '/lib/gopherbot_v1'

bot = Robot.new()

defaultConfig = <<'DEFCONFIG'
Channels:
- botdev
AllowDirect: false
Help:
- Keywords: [ "format", "world" ]
  Helptext: [ "(bot), format world - exercise formatting options" ]
CommandMatchers:
- Regex: '(?i:format world)'
  Command: "format"
DEFCONFIG

command = ARGV.shift()

case command
when "configure"
	puts defaultConfig
	exit
when "format"
    bot = bot.MessageFormat("Variable")
    proto = bot.GetBotAttribute("protocol")
    bot.Log("Audit", "Got: #{proto}")
    bot.Say("Hello, #{proto} World!")
    bot.Say('_italics_ <one> *bold* `code` @parsley')
    bot.Say('_italics_ <one> *bold* `code` @parsley', "raw")
    bot.Say('_italics_ <one> *bold* `code` @parsley', "variable")
    bot.Say('_italics_ <one> *bold* `code` @parsley', "fixed")
    bot.Say('_italics_ <one> *bold* `code` @parsley', "bogus")
end