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
* Multi-lingual with support for command plugins in Ruby, Python, PowerShell and Bash:
  * Most plugins can be distributed as a single text file
  * Plugins use blocking APIs vs registering callback functions, for easy scripting with a low learning curve
  * Plugin API includes WaitForReply* methods for simple question & answer flows
  * Flexible configuration can be easily overridden by the 'bot administrator
  * Configuration can limit plugins to certain channels and/or users
  * Sample plugins: [Ruby](plugins/rubydemo), [Python](plugins/pythondemo.py), [Bash](plugins/bashdemo), [PowerShell](plugins/psdemo.ps1)
* Configurable [elevated privilege](doc/Elevation.md) support for security-sensitive commands
* Localhost http/json interface for creating plugin APIs for other languages
* A thread-safe brain for storing state
* Configurable logging
* A supply of plugins and commands for viewing the log, reloading configuration, and more
* Many sample plugins, including knock-knock jokes!

## Development Status
Gopherbot is running in production in my environment. Currently lacking:
* Comprehensive plugin documentation
* Connectors for protocols other than Slack
* Brain implementations other than simple local files (e.g. redis)

## Documentation
* [Quick Start - Linux and Mac](doc/Quick-Start-Linux-Mac.md)
* [Quick Start - Windows](doc/Quick-Start-Windows.md)
* [Installing on Linux](doc/Linux-Install.md)
* [Installing on Windows](doc/Windows-Install.md)
* [Design Philosophy](doc/Design.md)
* [Configuration](doc/Configuration.md)
