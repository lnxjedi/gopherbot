## Quick Start

These instructions can be used to quickly set up and try out Gopherbot

1. Download and unzip the install archive in e.g. `<home directory>/gopherbot`
1. Copy `conf/gopherbot.yaml.sample` to `conf/gopherbot.yaml`
1. Obtain a 'bot token from https://\<your-team\>.slack.com/services/new/bot
1. Edit `gopherbot.yaml`, uncommenting and updating yaml lines for at least `AdminUsers`, `DefaultChannels` and `Protocol` configuration
1. From the install directory, run `./gopherbot` or `./gopherbot.exe`
   * `gopherbot(.exe) -h` will give you a list of command-line options
6. Once the 'bot is connected, invite your 'bot to `#general`, then type `help` and the 'bot will introduce itself
7. Press `ctrl-c` or tell the robot to `quit` to exit

### Optional - Enable other Plugins
* Enable other external plugins by adding them to `ExternalPlugins` in `gopherbot.yaml`, e.g.:
```yaml
ExternalPlugins:
- Name: weather
  Path: plugins/weather.rb
```
* To enable other built-in plugins, copy and edit the sample config in `conf/plugins`; most will need at least:
```yaml
Disabled: false
```
* Several optional plugins like `weather` require additional configuration; refer to the sample configurations in `conf/plugins`
