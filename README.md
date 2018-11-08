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

A few of Gopherbot's features for the upcoming 2.0:
* API support for pipelines and CI/CD operation (see [gopherci](jobs/gopherci.py))
* Scheduled job support

Features since 1.x:
* Built-in support for [elevated commands](doc/Security-Overview.md#elevation) requiring MFA (ala 'sudo')
* A prompting API for interactive plugins
* Comprehensive set of administrative commands
* Simple single-file script plugins

## Documentation

***NOTE: The documentation is currently incomplete and being updated for version 2.x. Documentation for older versions can be found in the `doc/` subdirectory of the distribution archive.***

Other than commented [configuration files](conf/gopherbot.yaml), most documentation is in the [doc/](doc/) folder. Also check the [Documentation Index](doc/README.md).

## Sample Plugin with the Ruby API
```ruby
#!/usr/bin/ruby
require 'net/http'
require 'json'

# NOTE: A bot administrator should supply the API key with a DM:
# `store task parameter weather OWM_APIKEY=xxxx`. Defaults for
# DEFAULT_COUNTRY and TEMP_UNITS can be overridden in custom
# conf/plugins/weather.yaml

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
```

The goal is for Gopherbot to support multiple scripting languages with built-in security features, to make secure ChatOps functions easy to write for most Systems and DevOps engineers.

Examples of work Gopherbot is already doing:
* Building and publishing new versions/snapshots of Gopherbot and [Luminos](https://github.com/lnxjedi/luminos)
* Updating development and production websites as commits are made to dev and merged to master
* Disabling / reenabling user web directories when requested by security engineers
* Adding users and updating passwords on servers using Ansible

## Development Status
Gopherbot 1.x is stable and running in production in several environments, and probably will not see further point updates.

Most of the work for 2.x is complete, with a release expected in the next 2-4 months.

### Contributing
See the [development notes](DevNotes.md) for design notes and stuff still needing to be done. Issues and pull requests welcome!

For development, testing, and collaboration, feel free to shoot me an email for an invite to [the LinuxJedi Slack team](https://linuxjedi.slack.com).
