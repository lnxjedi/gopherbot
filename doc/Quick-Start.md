## Quick Start, Linux and Mac

These instructions can be used to quickly set up a development environment for Gopherbot, where the robot runs as your user and all configuration and plugins reside in your home directory.

1. Download and unzip the install archive in `$HOME/gopherbot`
1. Create `$HOME/.gopherbot` for local configuration files
1. Copy the `brain/` and `conf/` directories from `$HOME/gopherbot` (install directory) to `$HOME/.gopherbot` (config directory)
1. Rename the installed `$HOME/gopherbot/conf/gopherbot.yaml.sample` to `gopherbot.yaml`
1. Obtain a 'bot token from https://\<your-team\>.slack.com/services/new/bot
1. Edit `$HOME/gopherbot/conf/gopherbot.yaml`, uncommenting and updating at least the AdminUsers, DefaultChannels and Protocol configuration
1. Run `$HOME/gopherbot/gopherbot`
1. Once the 'bot is connected, invite your 'bot to `#general`, then type `help` and the 'bot will introduce itself

`gopherbot -h` will give you a list of command-line options
