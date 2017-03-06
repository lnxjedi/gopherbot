# Gopherbot
## A.K.A. SUDO for the Enterprise
Gopherbot is:
* A chatops-oriented chat bot originally inspired by Hubot
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
* Simple scripting of command plugins in Ruby or Bash (with a Python library forthcoming)
 * [Sample Ruby Plugin](plugins/rubydemo)
 * [Sample Bash Plugin](plugins/bashdemo)
* Configurable [elevated privilege](doc/Elevation.md) support for security-sensitive commands
* Localhost http/json interface for creating plugin APIs for other languages
* Highly-configurable plugins that can be constrained by channels, users, etc.
* A thread-safe brain for storing state
* Configurable [logging](doc/Builtins#log)
* A supply of [built-in](doc/Builtins) plugins and commands for viewing the log, reloading configuration, and more 
* WaitForReply* methods for easily creating interactive plugins without complex session/state logic
* Many sample plugins, including knock-knock jokes!

## Development Status
Gopherbot is running in production in my environment. Currently lacking:
* Comprehensive documentation for Bash and Ruby plugin APIs
* Connectors for protocols other than Slack
* Brain implementations other than simple local files (e.g. redis)

## Documentation
* [Downloading and Installing](doc/Install.md)
* [Design Philosophy](doc/Design.md)
* [Advanced Installation and Configuration](doc/Configure.md)
* [Gopherbot Plugins](doc/Plugins.md)
* [Built-in Plugins](doc/Builtins.md)
* [Go Plugin API](doc/GoPlugins.md)
* [Ruby Plugin API](doc/RubyPlugins.md)
* [Bash Plugin API](doc/BashPlugins.md)
* [Security Considerations](doc/Security.md)
* [Configuring Elevation](doc/Elevation.md)
