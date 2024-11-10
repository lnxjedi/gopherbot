#!/usr/bin/env python3

# updatecfg.py - clone or update the robot's config repository.

# With Gopherbot 2.0, there's a strong assumption that custom
# configuration for the robot (jobs, plugins, yaml files) are stored
# in a git repository specified in the GOPHER_CUSTOM_REPOSITORY
# environment variable, which translates to CUSTOM_REPOSITORY_URL
# in the job (see the definition for the updatecfg job in
# conf/robot.yaml). When this job is run, the robot will attempt
# to clone or pull it's configuration repository.

# Note that if your config repo has a '.gopherci/pipeline.sh', it'll
# get executed - useful for e.g. installing $HOME/.ssh/config.

import os
from gopherbot_v2 import Robot

bot = Robot()

clone_url = os.getenv("GOPHER_CUSTOM_REPOSITORY")
clone_branch = os.getenv("GOPHER_CUSTOM_BRANCH")
cfgdir = os.getenv("GOPHER_CONFIGDIR")

if not clone_url:
    bot.Say("GOPHER_CUSTOM_REPOSITORY not set")
    exit()

if not cfgdir:
    bot.Say("GOPHER_CONFIGDIR not set")
    bot.Log("Error", "GOPHER_CONFIGDIR not set in updatecfg.py")
    exit()

if not bot.Exclusive("updatecfg", False):
    bot.Say("Configuration update already in progress")
    bot.Log("Warn", "Configuration update already in progress, exiting")
    exit()

bot.SetWorkingDirectory(cfgdir)
bot.FailTask("tail-log", [])
bot.FailTask("status", [ "Updating configuration failed"])

sshkey = os.getenv("KEYNAME")
# Add the ssh-agent task before git operations
bot.AddTask("ssh-agent", ["start", "ssh/%s" % sshkey])
# Add ssh-git-helper task to handle host key verification
host_keys = os.getenv("GOPHER_HOST_KEYS")
insecure_clone = os.getenv("GOPHER_INSECURE_CLONE") == "true"

if host_keys:
    bot.AddTask("ssh-git-helper", ["addhostkeys", host_keys])
else:
    bot.SetParameter("GOPHER_INSECURE_CLONE", "true" if insecure_clone else "false")
    bot.AddTask("ssh-git-helper", ["loadhostkeys", clone_url])
bot.AddTask("exec", [ "git", "pull" ])
bot.AddTask("status", [ "Custom configuration repository successfully updated" ])
bot.AddCommand("builtin-admin", "reload")
