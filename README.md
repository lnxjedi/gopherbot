# Gopherbot

[![CircleCI](https://circleci.com/gh/lnxjedi/gopherbot.svg?style=shield)](https://circleci.com/gh/lnxjedi/gopherbot)
[![GoDoc](https://godoc.org/github.com/lnxjedi/gopherbot/bot?status.png)](https://godoc.org/github.com/lnxjedi/gopherbot/bot)

Enterprise Slack(\*) ChatOps bot for Linux and Windows, supporting plugins in Python, Ruby, Bash and PowerShell

![](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/botdemo.gif)

(*) with a modular interface for writing other protocol connectors in Go

A few of Gopherbot's features:
* Built-in support for requiring MFA for select commands (ala 'sudo')
* A prompting API for interactive plugins
* Simple API for plugins in multiple scripting languages

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

The goal is for Gopherbot to support multiple scripting languages with built-in security features, to make ChatOps functions easy to write for most systems and DevOps engineers.

Examples of work Gopherbot is already doing:
* Building, backing up, restoring, stopping and starting instances in the AWS cloud
* Disabling / reenabling user web directories when requested by security engineers
* Adding users and updating passwords on servers using Ansible

## Development Status
Gopherbot is stable in production in my environment. Currently lacking:
* Comprehensive plugin API documentation
* Connectors for protocols other than Slack (and the trivial terminal/console connector)
* Brain implementations other than simple local files (e.g. redis)

### Contributing
Check out [Github's documentation](https://help.github.com/articles/fork-a-repo/) on forking and creating pull requests.

## Documentation
* [Quick Start - Linux and Mac](doc/Quick-Start-Linux-Mac.md)
* [Quick Start - Windows](doc/Quick-Start-Windows.md)
* [Installing on Linux](doc/Linux-Install.md)
* [Installing on Windows](doc/Windows-Install.md)
* [Design Philosophy](doc/Design.md)
* [Configuration](doc/Configuration.md)
* [Plugin Author's Guide](doc/Plugin-Author's-Guide.md)
