# Gopherbot
A chatops-oriented chatbot written in Go, inspired by Hubot.

## Features
* A thread-safe brain with consistency guarantees
* TOTP support for security-sensitive commands
* Highly-configurable plugins that can be constrained by channels, users, etc.
* Support for plugins written in Go, Ruby and Bash
* localhost http/json interface for creating plugin APIs for other languages
* WaitForReply API for easily creating interactive plugins without complex session/state logic
* Includes many sample plugins, including knock-knock jokes!

## Development Status
Gopherbot is running in production in my environment. Currently lacking:
* Comprehensive documentation for Bash and Ruby plugin APIs
* Connectors for protocols other than Slack
* Brain implementations other than simple local files (e.g. redis)

## Getting Gopherbot
### Binary releases
Pre-compiled archives for Linux and Mac OS are available for download directly from github:
https://github.com/uva-its/gopherbot/releases
### Installing from source
If you have a Go environment set up on your machine, you can can just `go get github.com/uva-its/gopherbot`

## Quick Start
1. Once you have a binary, obtain a 'bot token from https://yourteam.slack.com/services/new/bot
1. Copy `example.gopherbot` to `~/.gopherbot`
2. Edit `conf/gopherbot.yaml`, updating at least the AdminUsers and SlackToken
3. Run `./gopherbot` or $GOPATH/bin/gopherbot (depending on how you obtained Gopherbot, above)
4. Once the 'bot is connected, /invite it to any channels where it should be active (required for Slack)

`gopherbot -h` will give you a list of command-line options

## Included plugins

### Built-in plugins
Built-in plugins are always compiled in, and provide core/privileged functionality.
#### The help system
To get general help for your bot, simply type 'help' by itself in a channel. The 'bot will send you a private message giving details on using the help system and addressing the robot.

#### Administrator commands
The following administrator commands will be of some use:
* **reload** - Have the robot reload configuration, can be used to add a new plugin without a restart, or simply to pick up configuration changes
* **quit** - tell the robot to exit

These commands are only available by direct messaging the robot:
* **list plugins** - list all configured plugins
* **dump plugin (default) \<plugname\>** - show the current (or default) configuration for a plugin
* **dump (default) robot** - show the current (or default) configuration for the robot

### Compiled-in Plugins
Gopherbot includes a number of Go plugins that are compiled in by default; these include the **knock** plugin for telling knock-knock jokes, the **ping** plugin with several commands including 'ping' which just indicates if the robot is alive (or can here you). These can be disabled by setting `Disabled: true`; see **Configuring plugins** below.


## Installation Details
### Installdir
When Gopherbot starts, it first looks for it's install directory, containing at least `lib/` (plugin API libraries for Bash and Ruby), and `plugins/` (default/sample plugins shipped with Gopherbot). Gopherbot will check for the following directories:
* An installdir passed with the `-i` flag
* The location specified in the `$GOPHER_INSTALLDIR` environment variable, if set
* `/usr/local/share/gopherbot`
* `/usr/share/gopherbot`
* `$GOPATH/src/github.com/uva-its/gopherbot`, normally used when installing from source
* The directory where the gopherbot executable resides; this is normally used when running straight from the .zip archive

You can copy `lib/` and `plugins/` to one of the directories listed above (mainly for packaging/deployment), but for development and testing you can run gopherbot from the expanded archive directory and it'll look for it's install files there.

### Localdir
Gopherbot also looks for a local configuration directory, which must contain at least `conf/`, and optionally `plugins/` for your own plugins. You can copy the `example.gopherbot/` directory to one of the following locations that Gopherbot searches for local configuration:
* A config dir passed with the `-c` flag
* The location specified in the `$GOPHER_LOCALDIR` environment variable, if set
* `$HOME/.gopherbot`
* `/usr/local/etc/gopherbot`
* `/etc/gopherbot`

You'll need to edit the example config and at least set a correct SlackToken (obtainable from https://yourteam.slack.com/services/new/bot), but you should probably add your username to AdminUsers to enable e.g. the `reload` command.

### Configuring plugins
The 'bot administrator can override default plugin configuration by putting a new `<pluginname>.yaml` file in conf/plugins subdirectory of the local config dir. Top-level settings in this file will override the default configuration for a plugin. Note that if you override **Help**, **CommandMatches** or any other multi-part section, you should probably start by copying the entire section from the default config, as your settings will override the entire section.

See `example.gopherbot/conf/plugins/example.yaml` for a full example of a plugin configuration file.

## Writing your own plugins
The Go plugin API is fairly well documented here: http://godoc.org/github.com/uva-its/gopherbot/bot#Robot

The Bash and Ruby APIs aren't documented so well, however:
* Gopherbot includes several bash plugin examples and a fairly comprehensive `rubydemo` example
* The API libraries in `lib/` are fairly short and readble
* The Bash and Ruby APIs match the Go plugin API as closely as possible

### External Plugins
An external plugin is encapsulated in a single script, and added to the robot by adding a stanza to ExternalPlugins that gives the plugin name and path. The path can be absolute or relative; if relative, gopherbot checks Localdir then Installdir, so that a bot admin can modify a distributed plugin by simply copying it to `<localdir>/plugins` and editing it.

#### Structure
External plugins start with a bit of boilerplate code that loads the API library. This library is
responsible for translating function/method calls to http/json requests, and decoding the responses.

Once the library is loaded, ARGV[0] will contain a string command, optionally followed by arguments described below.

If the command is "configure", the plugin should dump it's default configuration to stdout; this is a yaml-formatted configuration that is common across all plugins. See `example.gopherbot/conf/plugins/example.yaml`.

If the command is "init", the plugin can perform any start-up actions, such as setting timers for automated messages. **NOTE:** This is currently disabled pending a bugfix.

**TODO: finish plugin writing documentation**
