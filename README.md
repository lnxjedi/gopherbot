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
https://github.com/parsley42/gopherbot/releases
### Installing from source
If you have a Go environment set up on your machine, you can can just `go get github.com/parsley42/gopherbot`

## Quick Start
1. Once you have a binary, obtain a 'bot token from https://yourteam.slack.com/services/new/bot
1. Copy `example.gopherbot` to `~/.gopherbot`
2. Edit `conf/gopherbot.yaml`, updating at least the AdminUsers and SlackToken
3. Run `./gopherbot` or $GOPATH/bin/gopherbot (depending on how you obtained Gopherbot, above)

`gopherbot -h` will give you a list of command-line options

## Installation Details
### Installdir
When Gopherbot starts, it first looks for it's install directory, containing at least `lib/` (plugin API libraries for Bash and Ruby), and `plugins/` (default/sample plugins shipped with Gopherbot). Gopherbot will check for the following directories:
* An installdir passed with the `-i` flag
* The location specified in the `$GOPHER_INSTALLDIR` environment variable, if set
* `/usr/local/share/gopherbot`
* `/usr/share/gopherbot`
* `$GOPATH/src/github.com/parsley42/gopherbot`, normally used when installing from source
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
