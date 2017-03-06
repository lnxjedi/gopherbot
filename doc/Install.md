## Getting Gopherbot

### Binary releases
1. Pre-compiled archives for Linux and Mac OS are available for download directly from github:
https://github.com/uva-its/gopherbot/releases
2. Unzip the archive in an install directory, e.g. `$HOME/gopherbot`

### Installing from source
If you have a Go environment set up on your machine, you can can:
`go get github.com/uva-its/gopherbot`


## Quick Start, Linux and Mac

These instructions can be used to quickly set up a development environment for Gopherbot, where the robot
runs as your user and all configuration and plugins reside in your home directory.

1. Create `$HOME/.gopherbot`
2. Copy the `brain/` and `conf/` directories to `$HOME/.gopherbot` from the install directory, or $GOPATH/src/github.com/uva-its/gopherbot if installed from source
1. Obtain a 'bot token from https://<your-team>.slack.com/services/new/bot
2. Edit `conf/gopherbot.yaml`, uncommenting and updating at least the AdminUsers, DefaultChannels and Protocol configuration
3. Run `./gopherbot` from your install directory or `$GOPATH/bin/gopherbot`
4. Once the 'bot is connected, invite your 'bot to `#general`, then type `help` and the 'bot will introduce itself

`gopherbot -h` will give you a list of command-line options