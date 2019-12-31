#!/usr/bin/env python3

# updatecfg.py - clone or update the robot's config repository.

# With Gopherbot 2.0, there's a strong assumption that custom
# configuration for the robot (jobs, plugins, yaml files) are stored
# in a git repository specified in the GOPHER_CUSTOM_REPOSITORY
# environment variable, which translates to CUSTOM_REPOSITORY_URL
# in the job (see the definition for the updatecfg job in
# conf/gopherbot.yaml). When this job is run, the robot will attempt
# to clone or pull it's configuration repository.

# Note that if your config repo has a '.gopherci/pipeline.sh', it'll
# get executed - useful for e.g. installing $HOME/.ssh/config.

import os
import re
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
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

bot.FailTask("status", [ "Updating configuration failed, check history for 'updatecfg'"])

if not clone_url.startswith("http"):
    match = re.match(r"ssh://(?:.*@)?([^:/]*)(?::([^/]*)/)?", clone_url)
    if match:
        bot.AddTask("ssh-init", [])
        scanhost = match.group(1)
        if match.group(2):
            scanhost = "%s:%s" % ( scanhost, match.group(2) )
        bot.AddTask("ssh-scan", [ scanhost ])
    else:
        match = re.match(r"(?:.*@)?([^:/]*)", clone_url)
        if match:
            bot.AddTask("ssh-init", [])
            bot.AddTask("ssh-scan", [ match.group(1) ])

bot.AddTask("git-sync", [ clone_url, clone_branch, cfgdir, "true" ])
bot.AddTask("runpipeline", [])
bot.AddTask("status", [ "Custom configuration repository successfully updated" ])
bot.AddCommand("builtin-admin", "reload")
