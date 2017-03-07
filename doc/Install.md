# Getting Gopherbot

## Binary releases
Pre-compiled archives for Linux and Mac OS are available for download directly from github:
https://github.com/uva-its/gopherbot/releases

## Installing from source
If you have a Go environment set up on your machine, you can can `go get github.com/uva-its/gopherbot`. To create an install archive run `./mkdist.sh` in `$GOPATH/src/github.com/uva-its/gopherbot`.

# Installing Gopherbot

## Quick Start, Linux and Mac

These instructions can be used to quickly set up a development environment for Gopherbot, where the robot
runs as your user and all configuration and plugins reside in your home directory.

1. Unzip the install archive in `$HOME/gopherbot`
1. Create `$HOME/.gopherbot` for configuration files
2. Copy the `brain/` and `conf/` directories from `$HOME/gopherbot` (install directory) to `$HOME/.gopherbot` (config directory)
1. Obtain a 'bot token from https://\<your-team\>.slack.com/services/new/bot
2. Edit `$HOME/.gopherbot/conf/gopherbot.yaml`, uncommenting and updating at least the AdminUsers, DefaultChannels and Protocol configuration
3. Run `$HOME/gopherbot/gopherbot`
4. Once the 'bot is connected, invite your 'bot to `#general`, then type `help` and the 'bot will introduce itself

`gopherbot -h` will give you a list of command-line options

## Production Installation of Gopherbot with systemd

As `root`:

1. Create a `robot` system user with home directory `/opt/robot`:
```
useradd -r -d /opt/robot -m robot
```
2. Unzip the Gopherbot install archive in `/opt/gopherbot`
3. Create `/usr/local/etc/gopherbot`, copy the `conf/` and `brain/` directories there, and give the robot user read/write access to `brain/`:
```
[root@myhost gopherbot]# cd /opt/gopherbot/
[root@myhost gopherbot]# mkdir /usr/local/etc/gopherbot
[root@myhost gopherbot]# cp -a conf/ brain/ /usr/local/etc/gopherbot/
[root@myhost gopherbot]# chown -R robot:robot /usr/local/etc/gopherbot/brain/
```
4. If you haven't already, get a Slack token for your robot from https://\<your-team\>.slack.com/services/new/bot
5. Edit `/opt/gopherbot/conf/gopherbot.yaml`, uncommenting and/or modifying:
  * `AdminContact`
  * `DefaultChannels`
  * `AdminUsers`
  * `Alias`
  * `Protocol` and `ProtocolConfig`
6. Copy `/opt/gopherbot/misc/gopherbot.service` to `/etc/systemd/system`
7. Reload systemd and set Gopherbot to start automatically:
```
[root@myhost gopherbot]# systemctl daemon-reload
[root@myhost gopherbot]# systemctl start gopherbot
[root@myhost gopherbot]# systemctl enable gopherbot
Created symlink /etc/systemd/system/multi-user.target.wants/gopherbot.service → /etc/systemd/system/gopherbot.service.
[root@myhost gopherbot]# systemctl status gopherbot
● gopherbot.service - Gopherbot DevOps Chatbot
   Loaded: loaded (/etc/systemd/system/gopherbot.service; disabled; vendor preset: disabled)
   Active: active (running) since Tue 2017-03-07 09:31:21 EST; 381ms ago
...
```
