# Getting Gopherbot

## Binary releases
Pre-compiled archives for Linux and Mac OS are available for download directly from github:
https://github.com/lnxjedi/gopherbot/releases

## Installing from source
If you have a Go environment set up on your machine, you can can `go get github.com/lnxjedi/gopherbot`. To create an install archive run `./mkdist.sh` in `$GOPATH/src/github.com/lnxjedi/gopherbot`.

# Production Installation of Gopherbot with systemd

Note that strictly speaking, the configuration directory is optional, but recommended for separating credentials from code in a production deployment.

As `root`:

1. Create a `robot` system user with home directory `/opt/robot`:
```
useradd -r -d /opt/robot -m robot
```
2. Unzip the Gopherbot install archive in `/opt/gopherbot`
3. Create `/usr/local/etc/gopherbot`, copy the `conf/` and `brain/` directories there
4. Rename `/opt/gopherbot/conf/gopherbot.yaml.sample` to `gopherbot.yaml`
5. Give the robot user read/write access to `brain/`:
```
[root@myhost gopherbot]# cd /opt/gopherbot/
[root@myhost gopherbot]# mkdir /usr/local/etc/gopherbot
[root@myhost gopherbot]# cp -a conf/ brain/ /usr/local/etc/gopherbot/
[root@myhost gopherbot]# chown -R robot:robot /usr/local/etc/gopherbot/brain/
```
6. If you haven't already, get a Slack token for your robot from https://\<your-team\>.slack.com/services/new/bot
7. Edit `/opt/gopherbot/conf/gopherbot.yaml`, uncommenting and/or modifying:
  * `AdminContact`
  * `DefaultChannels`
  * `AdminUsers`
  * `Alias`
  * `Protocol` and `ProtocolConfig`
8. Copy `/opt/gopherbot/misc/gopherbot.service` to `/etc/systemd/system`
9. Reload systemd and set Gopherbot to start automatically:
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
