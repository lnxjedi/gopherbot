![Gophers+bot by Renee French, cropped, cc3.0](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/gopherbot.PNG)

[Gophers + Robot by Renee French (cropped) licensed under Creative Commons License 3.0](https://creativecommons.org/licenses/by/3.0/)

# Gopherbot DevOps Chatbot

[![CircleCI](https://circleci.com/gh/lnxjedi/gopherbot.svg?style=shield)](https://circleci.com/gh/lnxjedi/gopherbot)
[![Coverage Status](https://coveralls.io/repos/github/lnxjedi/gopherbot/badge.svg?branch=master&service=github)](https://coveralls.io/github/lnxjedi/gopherbot?branch=master)
[![GoDoc](https://godoc.org/github.com/lnxjedi/gopherbot/bot?status.png)](https://godoc.org/github.com/lnxjedi/gopherbot/bot)

Enterprise Slack(\*) DevOps / ChatOps / CI/CD bot for Linux, supporting plugins in Python, Ruby, and Bash

Slogans under consideration:
* **The Co-worker that Never Sleeps**
* **The Swiss-Army Chainsaw for DevOps**

## Latest Release
Download the current release from: https://github.com/lnxjedi/gopherbot/releases/latest

## Documentation
The latest documentation on Github Pages: https://lnxjedi.github.io/gopherbot

---

![](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/botdemo.gif)

(*) with a modular interface for writing other protocol connectors in Go

## Gopherbot Version 2.0 (Unreleased)

Documentation and tests for version 2 are not yet finished, but configuration and API interfaces have settled to the point that the 2.0 snapshots are worth a look. Though it is stable and running in production for me, 2.0 is mostly for current users that are willing to do some digging around until documentation is finished and it's had more time to ripen. There will be a few more breaking changes before a full release.

### New Features

The biggest driver for version 2 is to remove the need for a separate CI/CD system. While **Gopherbot** couldn't replace Jenkins for many shops, it's very lightweight and perfectly adequate for many automated build / test and scheduled job workloads. Gopherbot is already building, running tests, and publishing itself; `.circleci` is only kept around to publish coverage results in a CircleCI-specific way. For a peek at GopherCI pipelines, compare the contents of `.circleci` to `.gopherci`.

Incomplete list of features new in 2.0:
* API support for pipelines and CI/CD operation (see [gopherci](jobs/gopherci.py))
* Rocket.Chat adapter
* Scheduled job support
* Job histories
* [Encryption](doc/Security-Overview.md) for secrets and the robot's brain
* Configuration merging; maps are merged, arrays are concatenated
* New default configuration and a wide selection of included external (script) jobs, plugins and tasks
* Go template substitutions in configuration files for e.g. referencing environment variables or decrypting secrets
* An [Ansible Playbook](https://github.com/lnxjedi/ansible-role-gopherbot)
* Docker images - see [Clu](https://github.com/parsley42/clu-docker) for clues on how to launch a container
* Go loadable modules, enabling new Go plugins at runtime

### Features since 1.x:
* Built-in support for [elevated commands](doc/Security-Overview.md#elevation) requiring MFA (ala 'sudo')
* A prompting API for interactive plugins
* Comprehensive set of administrative commands
* Support for simple single-file script plugins

### Breaking Changes

#### BotInfo
You need to populate at least `BotInfo:UserName:` for your robot to recognize it's name. See the [default configuration](conf/gopherbot.yaml)

#### ExternalPlugins
ExternalPlugins were formerly a list, but are now a hash, so:
```yaml
ExternalPlugins:
- Name: chuck
  Path: plugins/chuck.rb
```
   becomes:
```yaml
ExternalPlugins:
  "chuck":
    Path: plugins/chuck.rb
```

### Stuff that will change / go away
Currently it's possible to list tasks or plugins in ScheduledJobs; eventually only jobs will be able to be scheduled.

### Deprecated and Unsupported Platforms
The Windows port has been removed; the only known use case is being replaced. **Gopherbot** should build on Darwin (Mac OS X), but since builds with module support won't cross-compile, archives are no longer being generated.

## Documentation

With Version 2 nearly feature complete, documentation has become a priority. Watch https://lnxjedi.github.io/gopherbot for (hopefully) frequent updates. One of the best sources of documentation are the configuration repositories for **Floyd** (the production 'bot that builds all releases) and **Clu** (the devel 'bot that runs on my laptop); they can be found at [Github](https://github.com/parsley42).

Other than commented [configuration files](conf/gopherbot.yaml), most documentation is in the [doc/src/](doc/src/) folder. Stuff in `doc/src/Outdated` is just that.

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

### Contributing
I've started playing around with [Gitpod](https://gitpod.io) for development, with good results. Sometime after the next batch of core updates, I'll work on stabilizing this again. For now, issues and pull requests welcome.

For development, testing, and collaboration, feel free to shoot me an email for an invite to [the LinuxJedi Slack team](https://linuxjedi.slack.com).
