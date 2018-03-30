## Quick Start

These instructions can be used to quickly set up and try out Gopherbot with Slack, and is also useful for setting up
a development environment for your production robot. **NOTE:** Gopherbot uses **YAML** markup for all it's configuration files.
If you're unfamiliar with YAML, you might want to search for a quick reference. If you're in a hurry, the main thing you need to
know is the blank space at the beginning of lines must be spaces, not tabs, and that alignment is important. For most of the
sample files distributed with Gopherbot, removing the single `#` to uncomment lines will give you valid YAML.

1. Download and unzip the install archive to e.g. `<home directory>/gopherbot`
1. Copy `conf/gopherbot.yaml.sample` to `conf/gopherbot.yaml`
1. Obtain a 'bot token from https://\<your-team\>.slack.com/services/new/bot
1. Edit `gopherbot.yaml`, uncommenting and updating yaml lines for at least `AdminUsers`, `DefaultChannels` (listing `general`, `random` and other channels), and `Protocol` configuration
1. From the install directory, run `./gopherbot` or `./gopherbot.exe`
   * `gopherbot(.exe) -h` will give you a list of command-line options
6. Once the 'bot is connected, invite your 'bot to `#general`, then type `help` and the 'bot will introduce itself, with information about sending commands to the robot
7. Pressing `ctrl-c` or telling the robot to `quit` will end the process

## Basic administration

Once your 'bot is up and running, open a private chat and verify it responds to administrator commands; try:
* `info`
* `reload`
* `help log`
* `set loglevel debug`
* `show log`
* `show log page 1`

### Optional - Enable other Plugins
* Enable other external plugins by adding them to `ExternalPlugins` in `gopherbot.yaml`, e.g.:
```yaml
ExternalPlugins:
- Name: weather
  Path: plugins/weather.rb
```
Note that the `weather` plugin is the only script plugin that's useful for anything other than a code sample. It needs
a API token from https://openweathermap.org - see the comments in the script.
* To enable other built-in plugins, copy and edit the sample config in `conf/plugins`; most will need at least:
```yaml
Disabled: false
```
* Several optional plugins require additional configuration like `weather`; refer to the sample configurations in `conf/plugins`
* To enable a plugin without quitting and restarting, open a private chat and issue the `reload` command

## Writing Your First Plugin

This short example demonstrates a simple "hello world" plugin, written in bash.
* Open an editor and create `plugins/hello.sh` in your install directory:
```
#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
Help:
- Keywords: [ "hello", "world" ]
  Helptext: [ "(bot), hello world - the usual first program" ]
CommandMatchers:
- Regex: '(?i:hello world)'
  Command: "hello"
EOF
}

case "$COMMAND" in
	"configure")
		configure
		;;
	"hello")
	    Say "Hello, World!"
	    ;;
esac
```
* Save the file and mark it executable
* Edit `conf/gopherbot.yaml` and add your plugin to the list of external plugins:
```yaml
ExternalPlugins:
- Name: hello
  Path: plugins/hello.sh
```
* In the `general` channel:
   * Issue a `reload` command
   * Send your 'bot the `hello world` command

For writing more interesting plugins, see the [Plugin Author's Guide](Plugin-Author's-Guide.md).