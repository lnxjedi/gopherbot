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
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley')
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley', "raw")
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley', "variable")
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley', "fixed")
    bot.Say('_Italics_ <One> *Bold* `Code` @parsley', "bogus")
end