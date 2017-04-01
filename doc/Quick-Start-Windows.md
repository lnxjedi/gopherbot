## Quick Start, Windows

These instructions can be used to quickly set up a development environment for Gopherbot, where the robot
runs as your user and all configuration and plugins reside in your home directory.

1. Download and unzip the install archive in `%USERPROFILE%\gopherbot` (install directory)
1. Create `%USERPROFILE\.gopherbot` for configuration files (config directory)
2. Copy the `brain` and `conf` directories from the install directory to the config directory
1. Obtain a 'bot token from https://\<your-team\>.slack.com/services/new/bot
2. Edit `%USERPROFILE%\.gopherbot\conf\gopherbot.yaml`, uncommenting and updating at least the AdminUsers, DefaultChannels and Protocol configuration; optionally uncomment `ExternalPlugins` and the `psdemo.ps1` stanza to enable the example PowerShell plugin
3. Open a command prompt or powershell window and `cd gopherbot`, then run `.\gopherbot.exe`
4. Once the 'bot is connected, invite your 'bot to `#general`, then type `help` and the 'bot will introduce itself

`gopherbot -h` will give you a list of command-line options
