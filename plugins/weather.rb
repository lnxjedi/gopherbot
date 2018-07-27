#!/usr/bin/ruby
require 'net/http'
require 'json'

# To install:
# 1) Copy this file to plugins/weather.rb
# 2) Enable in gopherbot.yaml like so:
#ExternalScripts:
#- Name: weather
#  Path: plugins/weather.rb
#  Description: A plugin using OpenWeatherMap to give the weather
#  Parameters:
#  - Name: OWM_APIKEY
#    Value: "<yourkey>" # or omit and use admin command "store parameter weather OWM_APIKEY=<yourkey>"
#  - Name: TEMP_UNITS
#    Value: imperial # or 'metric'
#  - Name: DEFAULT_COUNTRY
#    Value: 'us' # or other ISO 3166 country code

# load the Gopherbot ruby library and instantiate the bot
require ENV["GOPHER_INSTALLDIR"] + '/lib/gopherbot_v1'
bot = Robot.new()

defaultConfig = <<'DEFCONFIG'
Help:
- Keywords: [ "weather" ]
  Helptext: [ "(bot), weather in <city(,country) or zip code> - fetch the weather from OpenWeatherMap" ]
CommandMatchers:
- Command: weather
  Regex: '(?i:weather (?:in|for) (.+))'
DEFCONFIG

command = ARGV.shift()

case command
when "configure"
	puts defaultConfig
	exit
when "weather"
    location = ARGV.shift()
    location += ",#{ENV["DEFAULT_COUNTRY"]}" unless location.include?(',')
    uri = URI("http://api.openweathermap.org/data/2.5/weather?q=#{location}&units=#{ENV["TEMP_UNITS"]}&APPID=#{ENV["OWM_APIKEY"]}")
    d = JSON::parse(Net::HTTP.get(uri))
    if d["message"]
        bot.Say("Sorry: \"#{d["message"]}\", maybe try the zip code?")
    else
        w = d["weather"][0]
        t = d["main"]
        bot.Say("The weather in #{d["name"]} is currently \"#{w["description"]}\" and #{t["temp"]} degrees, with a forecast low of #{t["temp_min"]} and high of #{t["temp_max"]}")
    end
end