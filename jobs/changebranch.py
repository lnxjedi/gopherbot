#!/usr/bin/env python3

# changebranch.py - switch branches for development

import os, sys
from gopherbot_v2 import Robot

bot = Robot()

executable = sys.argv.pop(0)
branch = sys.argv.pop(0)

clone_url = os.getenv("GOPHER_CUSTOM_REPOSITORY")
cfgdir = os.getenv("GOPHER_CONFIGDIR")

if not clone_url:
    bot.Say("GOPHER_CUSTOM_REPOSITORY not set")
    exit()

if not cfgdir:
    bot.Say("GOPHER_CONFIGDIR not set")
    bot.Log("Error", "GOPHER_CONFIGDIR not set in changebranch.py")
    exit()

# Use the same lock for update and changing branches
if not bot.Exclusive("updatecfg", False):
    bot.Say("Configuration update already in progress")
    bot.Log("Warn", "Configuration update already in progress, exiting")
    exit()

bot.SetWorkingDirectory(cfgdir)
bot.FailTask("tail-log", [])

bot.AddTask("git-init", [ clone_url ])
# Get new branches
bot.AddTask("exec", [ "git", "pull" ])
# Switch to the branch
bot.AddTask("exec", [ "git", "checkout", branch ])
# Make sure we're on latest commit for the branch
bot.AddTask("exec", [ "git", "pull" ])
bot.AddCommand("builtin-admin", "reload")
