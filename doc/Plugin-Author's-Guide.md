Gopherbot's functionality can be easily extended by writing plugins in one of several different languages. A single plugin can provide:
 * One or more new commands the robot will understand
 * Elevation logic for providing extra assurance of user identity
 * Authorization logic for determining a user's rights to issue various commands

This article deals mainly with writing plugins in one of the scripting languages supported by Gopherbot, the most popular means for writing new command plugins. For writing native compiled-in plugins in Go, see `gopherbot/main.go` and the sample plugins in `goplugins/`. API documentation for Robot methods is available at:

https://godoc.org/github.com/lnxjedi/gopherbot/bot#Robot

Note that the script plugin API is implemented on top of the native Go API, so that document may also be of use for scripting plugin authors. The file `bot/http.go`, and the scripting libraries in `lib/` will illuminate the mapping from the script APIs to the native Go API.

Table of Contents
=================

  * [Plugin Loading and Precedence](#plugin-loading-and-precedence)
  * [Default Configuration](#default-configuration)
  * [Calling Convention](#calling-convention)
    * [Environment Variables](#environment-variables)
    * [Reserved Commands](#reserved-commands)
  * [Plugin Types and Calling Events](#plugin-types-and-calling-events)
    * [Command Plugins](#command-plugins)
    * [Authorization Plugins](#authorization-plugins)
    * [Elevation Plugins](#elevation-plugins)
  * [Using the Terminal Connector](#using-the-terminal-connector)
    * [Using the Plugin Debugger](#plugin-debugging)
  * [Getting Started](#getting-started)
    * [Starting from a Sample Plugin](#starting-from-a-sample-plugin)
    * [Using Boilerplate Code](#using-boilerplate-code)
      * [Bash Boilerplate](#bash-boilerplate)
      * [PowerShell Boilerplate](#powershell-boilerplate)
      * [Python Boilerplate](#python-boilerplate)
      * [Ruby Boilerplate](#ruby-boilerplate)
  * [The Plugin API](#the-plugin-api)

# Plugin Loading and Precedence
Gopherbot ships with a number of external script plugins in the `install` directory. These can be overridden by placing a plugin with the same filename in the local configuration directory.

# Default Configuration
Plugin configuration is fully documented in the [configuration](Configuration.md) article; you should be familiar with that document before beginning to write your own plugins.

On start-up and during a reload, the robot will run each external script plugin with an argument of `configure`. The plugin should respond by writing the plugin default configuration to standard out and exiting with exit code 0. When responding to `configure`, the plugin shouldn't initialize a robot object or make any API calls, as `configure` is called without setting robot environment variables.

# Calling Convention
The robot calls external plugins by creating a goroutine and exec'ing the external script with a set of environment variables. The external script uses the appropriate library for the scripting language to create a robot object from the environment. The script then examines it's command-line arguments to determine the type of action to take (normally a command followed by arguments to the command), and uses the library to make JSON-over-http calls for executing and returning results from methods. Depending on how the plugin is used, different kinds of events can cause external plugins to be called with a variety of commands and arguments. The most common means of calling an external plugin is for one of it's commands to be matched, or by matching a pattern in an ambient message (one not specifically directed to the robot).

There are two sources of information for an external plugin being called:
  * Environment Variables - these should generally only be referenced by the scripting library
  * Command Line Arguments - these should be used by the plugin to determine what to do

## Environment Variables
Gopherbot sets two primary environment variables of use to the plugin developer:
  * GOPHER\_CONFIGDIR - the directory where Gopherbot looks for it's configuration
  * GOPHER\_INSTALLDIR - the directory where the Gopherbot executable resides
  
In addition, the following two environment variables are set for every script plugin:
  * GOPHER\_USER - the username of the user who spoke to the robot
  * GOPHER\_CHANNEL - the channel the user spoke in (empty string indicates a direct message)

## Reserved Commands
The first argument to a plugin script is the **command**. In addition to the `configure` command, which instructs a plugin to dump it's default configuration to standard out, the following commands are reserved:
* `init` - After starting the connector and on reload, the robot will call external plugins with a command argument of `init`. Since all environment variables for the robot are set at that point, it would be possible to e.g. save a robot data structure that could be loaded and used in a cron job.
* `authorize` - The plugin should check authorization for the user and return `Success` or `Fail`
* `elevate` - The plugin should perform additional authentication for the user and return `Success` or `Fail`
* `event` - This command is reserved for future use with e.g. user presence change & channel join/leave events
* `catchall` - Plugins with `CatchAll: true` will be called for commands directed at the robot that don't match a command plugin. Normally these are handled by the compiled-in `help` plugin, but administrators could override that setting and provide their own plugin with `CatchAll: true`. Note that having multiple such plugins is probably a bad idea.


# Plugin Types and Calling Events

There are (currently) three different kinds of external plugin:
 * Command / Message Plugins - these are called by the robot in respond to messages the user sends
 * Authorization Plugins - these plugins encapsulate the logic for authorizing specific users to use specific commands, and are called by the robot during authorization processing
 * Elevation Plugins - these plugins perform some variety of multi-factor authentication for higher assurance of user identity, and are called by the robot during elevation processing

In addition, external plugins can call each other using the CallPlugin(plugin, command, args...) method, subject to the target plugin's `TrustedPlugins` setting.

## Command Plugins

A command plugin's configuration specifies `CommandMatchers` and `MessageMatchers` that associate regular expressions with plugin commands:
```yaml
MessageMatchers:
- Command: help
  Regex: '^(?i:help)$'
CommandMatchers:
- Regex: (?i:remember ([-\w .,!?:\/]+))
  Command: remember
  Contexts: [ "item" ]
```
Whenever a `CommandMatcher` regex matches a command given to the robot, or a `MessageMatcher` matches an ambient message, the robot calls the plugin script with the first argument being the matched `Command`, and subsequent arguments corresponding to the regex capture groups (which may in some cases be an empty string). Command plugins should normally exit with status 0 (bot.Normal), or non-zero for unusual error conditions that may require an administrator to investigate. The robot will notify the user whenever a command plugin exits non-zero, or when it emits output to STDERR.

## Authorization Plugins
To separate command logic from user authorization logic, Gopherbot supports the concept of an **authorization plugin**. The main `gopherbot.yaml` can define a specific plugin as the `DefaultAuthorizer`, and individual plugins can be configured to override this value by specifying their own `Authorizer` plugin. If a plugin lists any commands in it's `AuthorizedCommands` config item, or specifies `AuthorizeAllCommands: true`, then the robot will call the authorizer plugin with a command of `authorize`, followed by the following arguments:
 * The name of the plugin for which authorization is being requested
 * The optional value of `AuthRequire`, which may be interpreted as a group or role
 * The plugin command being called followed by any arguments passed to the command

Based on these values and the `User` and `Channel` values from the robot, the authorization plugin should evaluate whether a user/plugin is authorized for the given command and exit with one of:
 * bot.Succeed (1) - authorized
 * bot.Fail (2) - not authorized
 * bot.MechanismFail (3) - a technical issue prevented the robot from determining authorization

Note that exiting with `bot.Normal` (0) or other values will result in an error and failed authentication.

Additionally, authorization plugins may provide extra feedback to the user on `Fail` or `MechanismFail` so they can have the issue addressed, e.g. "Authorization failed: user not a member of group 'foo'". In some cases, however, authorization plugins may not have a full Gopherbot API library; they could be written in C, and thus not be able to interact with the user.

## Elevation Plugins
Elevation plugins provide the means to request additional authentication from the user for commands where higher assurance of identity is desired. The main `gopherbot.yaml` can specify an elevation plugin as the `DefaultElevator`, which can be overridden by a given plugin specifying an `Elevator`. When the plugin lists commands as `ElevatedCommands` or `ElevateImmediateCommands`, the robot will call the appropriate elevator plugin with a command of `elevate` and a first argument of `true` or `false` for `immediate`. The elevator plugin should interpret `immediate == true` to mean MFA is required every time; when `immediate != true`, successful elevation may persist for a configured timeout period.

Based on the result of the elevation determination, the plugin should have an exit status one of:
 * bot.Succeed (1) - elevation succeeded
 * bot.Fail (2) - elevation failed
 * bot.MechanismFail (3) - a technical issue prevented the robot from processing the elevation request

Note that exiting with `bot.Normal` (0) or other value will result in an error being logged and elevation failing.

Additionally, the elevation plugin may provide extra feedback to the user when elevation isn't successful to indicate the nature of the failure.

# Using the Terminal Connector
Interacting with your bot in a chat app might not always be convenient or fast; to simplify
testing and plugin development, **Gopherbot** includes a terminal connector that emulates
a chat service with multiple users and channels, with a sample
configuration in the `cfg/term/` directory. You'll probably want to copy the directory and modify
it for your own use (mainly configuring the plugins you're developing), but it can be used
by using the `-c <configdir>` option:
```
[gopherbot]$ ./gopherbot -c cfg/term/
2018/04/13 18:07:52 Initialized logging ...
2018/04/13 18:07:52 Starting up with local config dir: cfg/term/, and install dir: /home/user/go/src/github.com/lnxjedi/gopherbot
2018/04/13 18:07:52 Debug: Loaded installed conf/gopherbot.yaml
2018/04/13 18:07:52 Debug: Loaded configured conf/gopherbot.yaml
Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user
c:general/u:alice -> |ubob
Changed current user to: bob
c:general/u:bob -> ;ping
general: @bob PONG
c:general/u:bob -> |ualice
Changed current user to: alice
c:general/u:alice -> |crandom
Changed current channel to: random
c:random/u:alice -> ;quit
random: @alice Adios
[gopherbot]$
```

## Plugin Debugging
**Gopherbot** has a builtin command for plugin debugging that will send information about
a plugin in direct messages. You can see plugin debugging in action here with
the terminal connector:
```
[gopherbot]$ ./gopherbot
2018/04/15 19:45:04 Initialized logging ...
2018/04/15 19:45:04 Starting up with local config dir: /home/parse/.gopherbot, and install dir: /home/parse/go/src/github.com/lnxjedi/gopherbot
2018/04/15 19:45:04 Debug: Loaded installed conf/gopherbot.yaml
2018/04/15 19:45:04 Debug: Loaded configured conf/gopherbot.yaml
Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user
c:general/u:parse -> ;ruby me!
general: @parse Sorry, that didn't match any commands I know, or may refer to a command that's not available in this channel; try 'floyd, help <keyword>'
c:random/u:parse -> ;help debug
random: Command(s) matching keyword: debug
floyd, debug plugin <pluginname> - turn on debugging for the named plugin

floyd, stop debugging - turn off debugging
c:general/u:parse -> ;debug plugin rubydemo
(dm:parse): 2018/04/15 07:45:18 DEBUG rubydemo: Loaded default config from the plugin, size: 1417
(dm:parse): 2018/04/15 07:45:18 DEBUG rubydemo: No configuration loaded from installPath (/home/parse/go/src/github.com/lnxjedi/gopherbot/conf/plugins/rubydemo.yaml): open /home/parse/go/src/github.com/lnxjedi/gopherbot/conf/plugins/rubydemo.yaml: no such file or directory
(dm:parse): 2018/04/15 07:45:18 DEBUG rubydemo: Loaded configuration from configPath (/home/parse/.gopherbot/conf/plugins/rubydemo.yaml), size: 22
general: Debugging enabled for rubydemo
c:general/u:parse -> ;ruby me!
(dm:parse): 2018/04/15 07:45:25 DEBUG rubydemo: plugin is NOT visible
general: @parse Sorry, that didn't match any commands I know, or may refer to a command that's not available in this channel; try 'floyd, help <keyword>'
c:general/u:parse -> |crandom
Changed current channel to: random
c:random/u:parse -> ;ruby me!
(dm:parse): 2018/04/15 07:46:34 DEBUG rubydemo: Checking 7 command matchers against message: "ruby me!"
(dm:parse): 2018/04/15 07:46:34 DEBUG rubydemo: Not matched: (?i:bashecho ([.;!\d\w-, ]+))
(dm:parse): 2018/04/15 07:46:34 DEBUG rubydemo: Matched regex '(?i:ruby( me)?!?)', command: ruby
(dm:parse): 2018/04/15 07:46:34 DEBUG rubydemo: Running plugin with command 'ruby' and arguments: [ me]
random: Sure, David!
random: Waaaaaait a second... what do you mean by that?
(dm:parse): 2018/04/15 07:46:35 DEBUG rubydemo: Plugin finished with return value: Normal
c:random/u:parse -> ;stop debugging
(dm:parse): 2018/04/15 07:47:02 DEBUG rubydemo: Checking 7 command matchers against message: "stop debugging"
(dm:parse): 2018/04/15 07:47:02 DEBUG rubydemo: Not matched: (?i:bashecho ([.;!\d\w-, ]+))
(dm:parse): 2018/04/15 07:47:02 DEBUG rubydemo: Not matched: (?i:ruby( me)?!?)
(dm:parse): 2018/04/15 07:47:02 DEBUG rubydemo: Not matched: (?i:listen( to me)?!?)
(dm:parse): 2018/04/15 07:47:02 DEBUG rubydemo: Not matched: (?i:remember(?: (slowly))? ([-\w .,!?:\/]+))
(dm:parse): 2018/04/15 07:47:02 DEBUG rubydemo: Not matched: (?i:recall ?([\d]+)?)
(dm:parse): 2018/04/15 07:47:02 DEBUG rubydemo: Not matched: (?i:forget ([\d]{1,2}))
(dm:parse): 2018/04/15 07:47:02 DEBUG rubydemo: Not matched: (?i:check me)
random: Debugging disabled
```

# Getting Started
## Starting from a Sample Plugin
The simplest way for a new plugin author to get started is to:
* Disable the demo plugin for your chosen scripting language (if enabled) in `<config dir>/conf/gopherbot.yaml`
* Copy the demo plugin to `<config dir>/plugins/<newname>(.extension)`
* Enable your new plugin in `gopherbot.yaml` and give it a descriptive `Name`

## Using Boilerplate Code
Each supported scripting language has a certain amount of "boilerplate" code required in every command plugin; generally, the boilerplate code is responsible for:
* Loading the appropriate version of the Gopherbot library from `$GOPHER_INSTALLDIR/lib`
* Defining and providing the default config
* Instantiating a Robot object with a library call
Normally this is followed by some form of case / switch statement that performs different functions based on the contents of the first argument, a.k.a. the "command".

### Bash Boilerplate
```bash
#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
<yaml config document>
EOF
}

case "$COMMAND" in
	"configure")
		configure
		;;
...
esac
```
**NOTE:** Bash doesn't have an object-oriented API

### PowerShell Boilerplate
```powershell
#!powershell.exe
# -or-
#!C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe

# Stylistic, can be omitted; $cmdArgs is always a String[],
# but $Args turns into a String when you shift off the 2nd item
[String[]]$cmdArgs = $Args
Import-Module "$Env:GOPHER_INSTALLDIR\lib\gopherbot_v1.psm1"
$bot = Get-Robot

$config = @'
<yaml config document>
'@

$command, $cmdArgs = $cmdArgs

switch ($command)
{
  "configure" {
    Write-Output $config
    exit
  }
 ...
}
```

### Python Boilerplate
```python
#!/usr/bin/python

import os
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v1 import Robot

bot = Robot()

default_config = '''
<yaml config document>
'''

executable = sys.argv.pop(0)
command = sys.argv.pop(0)

if command == "configure":
    print default_config
...
```
### Ruby Boilerplate
```ruby
#!/usr/bin/ruby

# boilerplate
require ENV["GOPHER_INSTALLDIR"] + '/lib/gopherbot_v1'

bot = Robot.new()

defaultConfig = <<'DEFCONFIG'
<yaml config document>
DEFCONFIG

command = ARGV.shift()

case command
when "configure"
	puts defaultConfig
	exit
...
end
```
# The Plugin API

Gopherbot has a rich set of methods (functions) for interacting with the robot / user. Here we break down the API into sets of related functions:
* [Attribute Lookup Methods](Attribute-Retrieval-API.md) - for retrieving names, email addresses, etc.
* [Calling other Plugins](CallPlugin-API.md) - for one plugin to trigger another
* [Sending Messages and Replies](Message-Sending-API.md) - for sending messages to the users
* [Prompting for Input](Response-Request-API.md) - for getting replies from the user
* [Brain Methods and Memories](Brain-API.md) - for storing and retrieving long- and short-term memories
* [Utility Methods](Utility-API.md) - a collection of miscellaneous useful functions, like Pause() and Log()
