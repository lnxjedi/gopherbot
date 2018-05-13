![Gophers+bot by Renee French, cropped, cc3.0](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/gopherbot.PNG)

[Gophers + Robot by Renee French (cropped) licensed under Creative Commons License 3.0](https://creativecommons.org/licenses/by/3.0/)

# Gopherbot

[![CircleCI](https://circleci.com/gh/lnxjedi/gopherbot.svg?style=shield)](https://circleci.com/gh/lnxjedi/gopherbot)
[![Coverage Status](https://coveralls.io/repos/github/lnxjedi/gopherbot/badge.svg?branch=master&service=github)](https://coveralls.io/github/lnxjedi/gopherbot?branch=master)
[![GoDoc](https://godoc.org/github.com/lnxjedi/gopherbot/bot?status.png)](https://godoc.org/github.com/lnxjedi/gopherbot/bot)

Enterprise Slack(\*) ChatOps bot for Linux and Windows, supporting plugins in Python, Ruby, Bash and PowerShell

Download the current release for your platform from: https://github.com/lnxjedi/gopherbot/releases/latest

To try out **Gopherbot** for yourself and write your first plugin, see [the Quick Start Guide](doc/Quick-Start.md)

![](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/botdemo.gif)

(*) with a modular interface for writing other protocol connectors in Go

A few of Gopherbot's features:
* Built-in support for [elevated commands](doc/Security-Overview.md#elevation) requiring MFA (ala 'sudo')
* A prompting API for interactive plugins
* Comprehensive set of administrative commands
* Simple single-file script plugins

## Sample Plugin with the Ruby API
```ruby
#!/usr/bin/ruby
require 'net/http'
require 'json'

# To install:
# 1) Copy this file to plugins/weather.rb
# 2) Enable in gopherbot.yaml like so:
#ExternalPlugins:
#- Name: weather
#  Path: plugins/weather.rb
# 3) Put your configuration in conf/plugins/weather.yaml:
#Config:
#  APIKey: <your openweathermap key>
#  TemperatureUnits: imperial # or 'metric'
#  DefaultCountry: 'us' # or other ISO 3166 country code

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
    c = bot.GetPluginConfig()
    location = ARGV.shift()
    location += ",#{c["DefaultCountry"]}" unless location.include?(',')
    uri = URI("http://api.openweathermap.org/data/2.5/weather?q=#{location}&units=#{c["TemperatureUnits"]}&APPID=#{c["APIKey"]}")
    d = JSON::parse(Net::HTTP.get(uri))
    if d["message"]
        bot.Say("Sorry: \"#{d["message"]}\", maybe try the zip code?")
    else
        w = d["weather"][0]
        t = d["main"]
        bot.Say("The weather in #{d["name"]} is currently \"#{w["description"]}\" and #{t["temp"]} degrees, with a forecast low of #{t["temp_min"]} and high of #{t["temp_max"]}")
    end
end
```

The goal is for Gopherbot to support multiple scripting languages with built-in security features, to make secure ChatOps functions easy to write for most Systems and DevOps engineers.

Examples of work Gopherbot is already doing:
* Building, backing up, restoring, stopping and starting instances in the AWS cloud
* Disabling / reenabling user web directories when requested by security engineers
* Adding users and updating passwords on servers using Ansible

## Development Status
Gopherbot 1.x is stable and running in production in several environments.

Development has started for Gopherbot 2.x, which adds features for scheduled jobs and pipelines. See the [development notes](DevNotes.md).

### Contributing
Check out [Github's documentation](https://help.github.com/articles/fork-a-repo/) on forking and creating pull requests. Feel free to shoot me an email
for an invite to [the LinuxJedi Slack team](https://linuxjedi.slack.com).

## Documentation

***NOTE: This is the documentation for Gopherbot v1.1. For older versions, see the documentation included in the distribution archive.***

Also see the full [Documentation Index](doc/README.md)
