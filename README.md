# Gopherbot
## A.K.A. SUDO for the Enterprise
Gopherbot is:
* A chat bot designed for ChatOps originally inspired by Hubot
* A simple remote script execution agent accessible via Slack(*)
* An easy means of delegating complex functionality to a user base without writing yet another web application

 (*) with a modular interface for writing other protocol connectors in Go

The goal is for Gopherbot to support enough scripting languages and security features to make chat bots useful for
doing **_real work_**. Examples of work Gopherbot is already doing:
* Building, backing up, restoring, stopping and starting instances in the AWS cloud
* Disabling / reenabling user web directories when requested by security engineers
* Adding users and updating passwords on servers using Ansible

See [Design Philosophy](doc/Design.md)

## Features
* Multi-platform with builds for Linux, Windows and Mac
* Simple command plugins with the following features:
  * Most plugins can be distributed as a single text file
  * Commands can be written in Ruby or Bash (with a Python library coming in 2017, sooner if I get a PR)
  * Plugins use blocking APIs vs registering callback functions, for easy scripting with a low learning curve
  * Plugin API includes WaitForReply* methods for simple question & answer flows
  * Flexible configuration can be easily overridden by the 'bot administrator
  * Configuration can limit plugins to certain channels and/or users
  * [Sample Ruby Plugin](plugins/rubydemo)
  * [Sample Bash Plugin](plugins/bashdemo)
* Configurable [elevated privilege](doc/Elevation.md) support for security-sensitive commands
* Localhost http/json interface for creating plugin APIs for other languages
* Highly-configurable plugins that can be constrained by channels, users, etc.
* A thread-safe brain for storing state
* Configurable [logging](doc/Builtins.md#log)
* A supply of [built-in](doc/Builtins.md) plugins and commands for viewing the log, reloading configuration, and more
* Many sample plugins, including knock-knock jokes!

## Development Status
Gopherbot is running in production in my environment. Currently lacking:
* Comprehensive plugin documentation
* Connectors for protocols other than Slack
* Brain implementations other than simple local files (e.g. redis)

## Documentation
* [Quick Start - Linux and Mac](doc/Quick-Start-Linux-Mac.md)
* [Installing on Linux](doc/Linux-Install.md)
* [Design Philosophy](doc/Design.md)
* [Configuration](doc/Configure.md)
[//]:* [Gopherbot Plugins](doc/Plugins.md)
[//]:* [Built-in Plugins](doc/Builtins.md)
[//]:* [Go Plugin API](doc/GoPlugins.md)
[//]:* [Ruby Plugin API](doc/RubyPlugins.md)
[//]:* [Bash Plugin API](doc/BashPlugins.md)
[//]:* [Security Considerations](doc/Security.md)
[//]:* [Configuring Elevation](doc/Elevation.md)
