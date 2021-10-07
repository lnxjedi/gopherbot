#!/usr/bin/env python3

# bootstrap.py - bootstrap a robot from env vars

import os
import sys
from gopherbot_v2 import Robot

bot = Robot()

# Pop off the executable path
sys.argv.pop(0)

command = sys.argv.pop(0)
if command != "init":
    exit(0)

cfgdir = os.getenv("GOPHER_CONFIGDIR")
cfgconf = os.path.join(cfgdir, "conf")

hasconfig = True
try:
    os.stat(cfgconf)
except FileNotFoundError:
    hasconfig = False
except:
    bot.Log("Error", "Checking for %s: %s" % (cfgconf, sys.exc_info()[0]))
    exit(1)

# First, see if we're doing a restore
if hasconfig:
    try:
        os.stat(".restore")
        bot.AddJob("restore", [])
        exit(0)
    except FileNotFoundError:
        pass
    exit(0)

clone_url = os.getenv("GOPHER_CUSTOM_REPOSITORY")
if len(clone_url) == 0:
    bot.Log("Warn", "GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping")
    exit(0)

clone_branch = os.getenv("GOPHER_CUSTOM_BRANCH")

if not clone_url.startswith("http"):
    depkey = os.getenv("GOPHER_DEPLOY_KEY")
    if len(depkey) == 0:
        bot.Log("Error", "SSH required for bootstrapping and no GOPHER_DEPLOY_KEY set")
        exit(1)

bot.Log("Info", "Creating bootstrap pipeline for %s" % clone_url)
bot.SetParameter("BOOTSTRAP", "true")
bot.SetParameter("GOPHER_DEPLOY_KEY", depkey)
bot.AddTask("git-init", [ clone_url ])

tkey = os.path.join(cfgdir, "binary-encrypted-key")
bot.AddTask("exec", [ "rm", "-f", tkey ])
bot.AddTask("exec", [ "touch", ".restore" ])
bot.AddTask("git-clone", [ clone_url, clone_branch, cfgdir, "true" ])
bot.AddTask("run-pipeline", [])
bot.AddTask("restart-robot", [])
