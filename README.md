# Gopherbot

[![CircleCI](https://circleci.com/gh/lnxjedi/gopherbot.svg?style=shield)](https://circleci.com/gh/lnxjedi/gopherbot)
[![GoDoc](https://godoc.org/github.com/lnxjedi/gopherbot/bot?status.png)](https://godoc.org/github.com/lnxjedi/gopherbot/bot)

Enterprise Slack(\*) ChatOps bot for Linux and Windows, supporting plugins in Python, Ruby, Bash and PowerShell

![](https://raw.githubusercontent.com/wiki/lnxjedi/gopherbot/botdemo.gif)

A few of Gopherbot's features:
* Built-in support for MFA
* A prompting API for interactive plugins
* Simple API for plugins in multiple scripting languages

 (*) with a modular interface for writing other protocol connectors in Go

The goal is for Gopherbot to support enough scripting languages and security features to make chat bots useful for
a wider range of ops functions. Examples of work Gopherbot is already doing:
* Building, backing up, restoring, stopping and starting instances in the AWS cloud
* Disabling / reenabling user web directories when requested by security engineers
* Adding users and updating passwords on servers using Ansible

## Development Status
Gopherbot is stable in production in my environment. Currently lacking:
* Comprehensive plugin API documentation
* Connectors for protocols other than Slack
* Brain implementations other than simple local files (e.g. redis)

### Contributing
Check out [Github's documentation](https://help.github.com/articles/fork-a-repo/) on forking and creating pull requests. It's fun! Everybody is doing it!

## Documentation
* [Quick Start - Linux and Mac](doc/Quick-Start-Linux-Mac.md)
* [Quick Start - Windows](doc/Quick-Start-Windows.md)
* [Installing on Linux](doc/Linux-Install.md)
* [Installing on Windows](doc/Windows-Install.md)
* [Design Philosophy](doc/Design.md)
* [Configuration](doc/Configuration.md)
* [Plugin Author's Guide](doc/Plugin-Author's-Guide.md)
