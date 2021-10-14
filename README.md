# Gopherbot DevOps Chatbot

Slack(\*) DevOps / ChatOps bot for Linux, supporting extensions in Bash, Python, Ruby, and Go.

> (*) with a modular interface for writing other protocol connectors in Go

Slogans under consideration:
* **The Co-worker that Never Sleeps**
* **The DevOps Swiss-Army Chainsaw**

## What does Gopherbot *do*?
**Gopherbot** runs as a process or container in your infrastructure and connects to your team chat. From there it can respond to CLI-like commands written in Bash, Ruby or Python (\*), and perform any number of functions to provision resources, run reports and scheduled jobs, deploy software or interact with CI/CD - just about any functionality a DevOps engineer might want to provide in team chat. You can find a lot more information in the [introduction](https://lnxjedi.github.io/gopherbot/Introduction.html) of the online manual.

> (*) More advanced and Slack-specific extensions can be written in **Go**; support for additional scripting languages is relatively simple to add.

## Major Features
* Powerful pipeline-oriented engine for creating and combining reusable components in multiple scripting languages
* Flexible support for encrypted secrets
* Wide variety of security features including built-in Google Authenticator TOTP
* Full-featured terminal connector and built-in IDE for developing extensions
* Highly configurable with Go-templated YAML

## Software Overview
**Gopherbot** uses a model similar to [Ansible](https://ansible.com), distributing a core robot with an array of built-in reusable components. Individual robots are configured and stored in a **git** repository that is cloned when you bootstrap your robot in your infrastructure; several example robot repositories are given below.

Running a **Gopherbot** robot essentially means running the core robot (on a VM or in a container) with a handful of environment variables that tell the core robot how to clone and run *your* individual robot.

## Documentation
The latest documentation can always be found at the GitHub-hosted [online manual](https://lnxjedi.github.io/gopherbot); the documentation source is in a [separate repository](https://github.com/lnxjedi/gopherbot-doc). Documentation automatically generated from the Go sources can be found at [pkg.go.dev](https://pkg.go.dev/github.com/lnxjedi/gopherbot/v2).

The manual is still somewhat incomplete; however, sometimes the best documentation is example code. To that end, the most powerful and complete robot I have is [Mr. Data](https://github.com/parsley42/data-gopherbot)(now retired) - the robot that ran my home Kubernetes cluster when I still had time for such things. [Clu](https://github.com/parsley42/clu-gopherbot) is the development robot used for development and writing documentation. Though **Clu** doesn't do any useful work, he has examples of most facets of **Gopherbot** functionality. [Floyd](https://github.com/parsley42/floyd-gopherbot) (a utility robot I share with my wife) is the oldest and longest-running robot instance - occasionally he does useful work, but mostly he just makes dinner meal suggestions.

## Release Status
Version 2 has been stable for me for over a year, and has finally been released. I've accepted that a fully up-to-date manual will lag significantly, but that is currently where the most work is being done.

With the recent (2021) addition of `ParameterSets` and a container-based **IDE**, there are no major updates in functionality currently planned.

## Previewing
If you have [Docker](https://www.docker.com/) available, you can kick the tires on the default robot running the **terminal** connector:
```
$ docker run -it quay.io/lnxjedi/gopherbot
...
Terminal connector running; Type '|c?' to list channels, '|u?' to list users
...
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration was
detected, you're connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try
'gopherbot --help'; otherwise feel free to play around with the default robot - you
can start by typing 'help'. If you'd like to start configuring a new robot, type:
';setup slack'.
```

## Downloading
You can download the latest release build from the [release page](https://github.com/lnxjedi/gopherbot/releases/latest). Container builds can be found on [Quay.IO](https://quay.io/organization/lnxjedi).

### Gopherbot Container Variants
Each release of **Gopherbot** creates three container variants:
* [gopherbot](https://quay.io/repository/lnxjedi/gopherbot) - `quay.io/lnxjedi/gopherbot`
  * `gopherbot` is a fairly minimal gopherbot container for running a containerized robot
* [gopherbot-dev](https://quay.io/repository/lnxjedi/gopherbot-dev) - `quay.io/lnxjedi/gopherbot-dev`
  * `gopherbot-dev` uses [Theia](https://github.com/theia-ide/theia-apps) for the entrypoint, and is intended for use in setting up a new robot
* [gopherbot-theia](https://quay.io/repository/lnxjedi/gopherbot-theia) - `quay.io/lnxjedi/gopherbot-theia`
  * `gopherbot-theia` includes theia but uses `gopherbot` for the entrypoint; this is the official **Gopherbot IDE** for developing extensions

## Building from Source
Building from source is straight-forward as `make dist` with the `Makefile`, as long as the build system has all the requirements.

**Requirements:**
* A recent (1.14+) version of Go
* Standard build utilities; make, tar, gzip

**Steps:**
1. Clone this repository
1. Optionally check out a release version - `git checkout v2.0`
1. `make dist` in the repository root to create an installable archive, or just `make` to build the binaries
1. Follow the [manual installation instructions](https://lnxjedi.github.io/gopherbot/install/ManualInstall.html) for installing an archive on your system

---

This example transcript is a little outdated, and doesn't showcase the new job functionality introduced in version 2 - but **Gopherbot** still knows how to tell jokes.

![](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/botdemo.gif)

## Deprecated and Unsupported Platforms
The Windows and Darwin (MacOS) ports have both been removed. The best solution for these platforms is to take advantage of the excellent Linux container support to run your robot in a container, perhaps with [Docker Desktop](https://www.docker.com/products/docker-desktop). [WSL](https://docs.microsoft.com/en-us/windows/wsl/install) is also a good solution for Windows.

## Sample Plugin with the Ruby API
```ruby
#!/usr/bin/ruby
require 'net/http'
require 'json'

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

# NOTE: the required environment variables need to be supplied as
# `Parameters` for the `weather` plugin in custom/conf/robot.yaml.
# The API key should be encrypted.

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
PR's welcome. For development, testing, and collaboration, feel free to shoot me an email for an invite to [the LinuxJedi Slack team](https://linuxjedi.slack.com).
