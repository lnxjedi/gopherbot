#!/usr/bin/ruby
require 'net/http'
require 'json'

# load the Gopherbot ruby library and instantiate the bot
require ENV["GOPHER_INSTALLDIR"] + '/lib/gopherbot_v1'
bot = Robot.new()

defaultConfig = <<'DEFCONFIG'
MessageMatchers:
- Command: chuck
  Regex: '(?i:chuck norris)'
Config:
  Openings:
  - "Chuck Norris?!?! He's AWESOME!!!"
  - "Oh cool, you like Chuck Norris, too?"
  - "Speaking of Chuck Norris - "
  - "Hey, I know EVERYTHING about Chuck Norris!"
  - "I'm a HUUUUGE Chuck Norris fan!"
  - "Not meaning to eavesdrop or anything, but are we talking about CHUCK NORRIS ?!?"
  - "Oh yeah, Chuck Norris! The man, the myth, the legend."
DEFCONFIG

command = ARGV.shift()

case command
when "configure"
	puts defaultConfig
	exit
when "chuck"
    uri = URI("http://api.icndb.com/jokes/random")
    d = JSON::parse(Net::HTTP.get(uri))
    opening = bot.RandomString(bot.GetTaskConfig()["Openings"])
    bot.Say("#{opening} Did you know ...?")
    bot.Pause(2)
    bot.Say(d["value"]["joke"])
end