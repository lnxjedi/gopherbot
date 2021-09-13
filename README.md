![Gophers+bot by Renee French, cropped, cc3.0](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/gopherbot.PNG)

[Gophers + Robot by Renee French (cropped) licensed under Creative Commons License 3.0](https://creativecommons.org/licenses/by/3.0/)

# Gopherbot DevOps Chatbot

Enterprise Slack(\*) DevOps / ChatOps / CI/CD bot for Linux, supporting extensions in Go, Python, Ruby, and Bash

> (*) with a modular interface for writing other protocol connectors in Go

Slogans under consideration:
* **The Co-worker that Never Sleeps**
* **The DevOps Swiss-Army Chainsaw**

## What does Gopherbot *do*?
**Gopherbot** logs in to your team chat like a normal user, and can respond to CLI-like commands for interacting with infrastructure. Since Gopherbot relies heavily on external shell utilities and scripting languages (python, ruby, bash) for adding functionality, it's strongest use case is kind of a "super **CRON**", running jobs on a schedule or on demand, and reporting any problems in your team chat. It can be used with strongly-encrypted secrets to run ansible playbooks or python/ruby/curl scripts interacting with remote APIs. Other use cases include manually triggering software deployments, or updating a status message on your company's website. If you have a collection of scripts for doing this kind of thing, and have ever wished you could trigger these scripts in your team chat, chances are good **Gopherbot** is the tool for you. You can find a lot more information in the [introduction](https://lnxjedi.github.io/gopherbot/Introduction.html) of the online manual.

## Architecture
**Gopherbot** uses a model similar to [Ansible](https://ansible.com), distributing a core robot with an array of built-in functionality. Individual robots are configured and stored in a **git** repository that is cloned during the robot bootstrap process; several example robot repositories are given below.

Running a **Gopherbot** robot essentially means running the core robot (on a VM or in a container) with a handful of environment variables that tell the core robot how to clone and run *your* individual robot.

## Documentation
The latest documentation can always be found at the GitHub-hosted [online manual](https://lnxjedi.github.io/gopherbot); the documentation source is in a [separate repository](https://github.com/lnxjedi/gopherbot-doc). Automatically generated documentation can be found at [pkg.go.dev](https://pkg.go.dev/github.com/lnxjedi/gopherbot/v2).

The manual is still somewhat incomplete; however, sometimes the best documentation is example code. To that end, the most powerful and complete robot I have is [Mr. Data](https://github.com/parsley42/data-gopherbot) - the robot that ran my home Kubernetes cluster when I still had time for such things. The repositories for [Floyd](https://github.com/parsley42/floyd-gopherbot) (a utility robot I share with my wife) and [Clu](https://github.com/parsley42/clu-gopherbot) (the devel 'bot that runs on my laptop) are also available.

## Release Status
Version 2 has been stable for me for over a year, and has finally been released. I've accepted that a fully up-to-date manual will lag significantly.

Future development will be focused on documentation, bugfixes and enhancements, with no major updates in functionality currently planned.

## Downloading
You can download the latest release build from the [release page](https://github.com/lnxjedi/gopherbot/releases/latest).

## Building from Source
Building from source is straight-forward as `make dist` with the `Makefile`, as long as the build system has all the requirements.

**Requirements:**
* A recent (1.14+) version of Go
* Standard build utilities; make, tar, gzip
> Note that a bug in Go v1.16 causes a crash on startup which should be fixed in the [next release](https://github.com/golang/go/issues/44586).

**Steps:**
1. Clone this repository
1. Optionally check out a release version - `git checkout v2.0`
1. `make dist` in the repository root to create an installable archive, or just `make` to build the binaries
1. Follow the [manual installation instructions](https://lnxjedi.github.io/gopherbot/install/ManualInstall.html) for installing an archive on your system

## Gopherbot Container Variants
Each release of **Gopherbot** creates three container variants:
* [gopherbot](https://quay.io/repository/lnxjedi/gopherbot) - `quay.io/lnxjedi/gopherbot`
  * `gopherbot` is a fairly minimal gopherbot container for running a containerized robot
* [gopherbot-dev](https://quay.io/repository/lnxjedi/gopherbot-dev) - `quay.io/lnxjedi/gopherbot-dev`
  * `gopherbot-dev` uses [Theia](https://github.com/theia-ide/theia-apps) for the entrypoint, and is intended for use in setting up a new robot
* [gopherbot-theia](https://quay.io/repository/lnxjedi/gopherbot-theia) - `quay.io/lnxjedi/gopherbot-theia`
  * `gopherbot-theia` includes theia but uses `gopherbot` for the entrypoint, intended for developing extensions for existing robots

---

This example transcript is a little outdated, and doesn't showcase the new job functionality introduced in version 2 - but **Gopherbot** still knows how to tell jokes.

![](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/botdemo.gif)

### Deprecated and Unsupported Platforms
The Windows and Darwin (MacOS) ports have both been removed. The best solution for these platforms is to take advantage of the excellent Linux container support to run your robot in a container, perhaps with [Docker Desktop](https://www.docker.com/products/docker-desktop).

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
require 'gopherbot_v1'
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
For development, testing, and collaboration, feel free to shoot me an email for an invite to [the LinuxJedi Slack team](https://linuxjedi.slack.com).
