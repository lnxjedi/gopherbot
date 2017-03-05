## Getting Gopherbot
### Binary releases
Pre-compiled archives for Linux and Mac OS are available for download directly from github:
https://github.com/uva-its/gopherbot/releases
### Installing from source
If you have a Go environment set up on your machine, you can can:
* `go get github.com/uva-its/gopherbot`
* Run `./mkdist.sh` in $GOPATH/src/github.com/uva-its/gopherbot to create a distribution `.zip` file

## Quick Start, Linux and Mac

These instructions can be used to quickly set up a development environment for Gopherbot, with all
configuration and plugins in the user's home directory.

1. Unzip the distribution to an install directory, e.g. `$HOME/gopherbot`
2. Create `$HOME/.gopherbot`, and copy the `/conf` and `/brain` directories there
3. Obtain a 'bot token from https://<yourteam>.slack.com/services/new/bot
2. Edit `conf/gopherbot.yaml`, updating at least the AdminUsers and Protocol configuration
3. Run `./gopherbot` from your install directory
4. Once the 'bot is connected, invite your 'bot to `#general`, then type `help` and the 'bot will introduce itself

`gopherbot -h` will give you a list of command-line options