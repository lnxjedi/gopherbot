This article deals mainly with writing plugins in one of the scripting languages supported by Gopherbot,
the most popular means for writing new command plugins. For writing native compiled-in plugins in Go, see
`gopherbot/main.go` and the sample plugins in `goplugins/`. API documentation for Robot methods is available
at:

https://godoc.org/github.com/uva-its/gopherbot/bot#Robot

Note that the script plugin API is implemented on top of the native Go API, so that document may also be of use for scripting plugin authors.

Table of Contents
=================

  * [Default Configuration](#default-configuration)
  * [Calling Convention](#calling-convention)
    * [Reserved Commands](#reserved-commands)
  * [Getting Started](#getting-started)
    * [Starting from a Sample Plugin](#starting-from-a-sample-plugin)
    * [Using Boilerplate Code](#using-boilerplate-code)
      * [Bash Boilerplate](#bash-boilerplate)
      * [PowerShell Boilerplate](#powershell-boilerplate)
      * [Python Boilerplate](#python-boilerplate)
      * [Ruby Boilerplate](#ruby-boilerplate)
  * [The Plugin API](#the-plugin-api)

# Default Configuration
Plugin configuration is fully documented in the [configuration](Configuration.md) article; you should be familiar with that document before beginning to write your own plugins.

On start-up and during a reload, the robot will run each external script plugin with an argument of `configure`. The plugin should respond by writing the plugin default configuration to standard out and exiting with exit code 0. When responding to `configure`, the plugin shouldn't initialize a robot object or make any API calls, as `configure` is called without setting robot environment variables.

# Calling Convention
The plugin's configuration specifies `CommandMatchers` and `MessageMatchers` that associate regular expressions with plugin commands:
```yaml
MessageMatchers:
- Command: help
  Regex: '^(?i:help)$'
CommandMatchers:
- Regex: (?i:remember ([-\w .,!?:\/]+))
  Command: remember
  Contexts: [ "item" ]
```
Whenever a `CommandMatcher` regex matches a command given to the robot, or a `MessageMatcher` matches an ambient message, the robot spawns a child process with a set of environment variables, and executes the plugin script with the first argument being the matched `Command`, and subsequent arguments corresponding to the regex capture groups (which may in some cases be an empty string). During script execution, plugins use a language-specific API to interact with the robot and the user. Normally this means instantiating the Robot object which script libraries construct from environment variables, and then calling methods on it.

## Reserved Commands
In addition to the `configure` command, which instructs a plugin to dump it's default configuration to standard out, the following commands are reserved:
* `init` - During startup and reload, the robot will call external plugins with an argument of `init`. Since all environment variables for the robot are set at that point, it would be possible to e.g. save a robot data structure that could be loaded and used in a cron job.
* `event` - This command is reserved for future use with e.g. user presence change & channel join/leave events

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
* [Message Sending Methods](Message-Sending-API.md) - for sending messages to the users
* [Attribute Retrieval Methods](Attribute-Retrieval-API.md) - for retrieving names, email addresses, etc.
* [Response Request Methods](Reponse-Request-API.md) - for getting replies from the user
* [Long-term Memory Methods](Long-term-Memory-API.md) - for storing long-term memories (like a TODO list, or user preference)
* [Short-term Memory Methods](Short-term-Memory-API.md) - for storing short-term memories like conversation context that are stored in memory and expire after a period of time
* [Security and Elevation Methods](Security-API.md) - for making determinations on privileged commands
* [Utility Methods](Utility-API.md) - a collection of miscellaneous useful functions, like Pause()
