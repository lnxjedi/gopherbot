# Gopherbot
Gopherbot is a chatops-oriented chat bot written in Go, inspired by Hubot. The goal is for
Gopherbot to support enough programming languages and security features to be useful for
doing **_real work_**. A sample of work Gopherbot is already doing:
* Building, backing up, starting and stopping instances in the AWS cloud
* Disabling / reenabling user web directories when requested by security engineers
* Adding users and updating passwords on servers using Ansible

## Features
* [Elevated privilege](doc/Elevation.md) support for security-sensitive commands
* Highly-configurable plugins that can be constrained by channels, users, etc.
* A thread-safe brain with consistency guarantees
* Support for Go plugins and external script plugins written in Ruby and Bash
* Localhost http/json interface for creating plugin APIs for other languages
* Configurable [logging](doc/Logging)
* A supply of [built-in](doc/Builtins) plugins and commands for viewing logs, reloading configuration, and more 
* WaitForReply* methods for easily creating interactive plugins without complex session/state logic
* Includes many sample plugins, including knock-knock jokes!

## Development Status
Gopherbot is running in production in my environment. Currently lacking:
* Comprehensive documentation for Bash and Ruby plugin APIs
* Connectors for protocols other than Slack
* Brain implementations other than simple local files (e.g. redis)

## Documentation
* [Design Philosophy](doc/Design.md)
* [Downloading and Installing](doc/Install.md)
* [Advanced Installation and Configuration](doc/Configure.md)
* [Goperbot Plugins](doc/Plugins.md)
* [Built-in Plugins](doc/Builtins.md)
* [Go Plugin API](doc/GoPlugins.md)
* [Ruby Plugin API](doc/RubyPlugins.md)
* [Bash Plugin API](doc/BashPlugins.md)
* [Security Considerations](doc/Security.md)
* [Configuring Elevation](doc/Elevation.md)
