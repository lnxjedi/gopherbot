#!/usr/bin/env python3

# bootstrap.py - bootstrap a robot from env vars

import os
import re
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v2 import Robot

bot = Robot()

# Pop off the executable path
sys.argv.pop(0)

command = sys.argv.pop(0)
if command != "init":
    exit(0)

# First, see if we're doing a restore
try:
    os.stat(".restore")
    bot.AddTask("exec", [ "rm", "-f", ".restore" ])
    bot.AddJob("restore", [])
    exit(0)
except FileNotFoundError:
    pass

cfgdir = os.getenv("GOPHER_CONFIGDIR")
cfgfile = os.path.join(cfgdir, "conf", "gopherbot.yaml")

try:
    os.stat(cfgfile)
except FileNotFoundError:
    pass
except:
    bot.Log("Error", "Checking for gopherbot.yaml: %s" % sys.exc_info()[0])
    exit(1)
else:
    exit(0)

clone_url = os.getenv("GOPHER_CUSTOM_REPOSITORY")
if len(clone_url) == 0:
    bot.Log("Warn", "GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping")
    exit(0)

clone_branch = os.getenv("GOPHER_CUSTOM_BRANCH")

if not clone_url.startswith("http"):
    depkey = os.getenv("DEPLOY_KEY")
    if len(depkey) == 0:
        bot.Log("Error", "SSH required for bootstrapping and no DEPLOY_KEY set")
        exit(1)

bot.Log("Info", "Creating bootstrap pipeline for %s" % clone_url)
bot.SetParameter("BOOTSTRAP", "true")
bot.SetParameter("DEPLOY_KEY", depkey)
bot.AddTask("git-init", [ clone_url ])

tkey = os.path.join(cfgdir, "binary-encrypted-key")
bot.AddTask("exec", [ "rm", "-f", tkey ])
bot.AddTask("exec", [ "touch", ".restore" ])
bot.AddTask("git-clone", [ clone_url, clone_branch, cfgdir, "true" ])
bot.AddTask("run-pipeline", [])
bot.AddTask("restart-robot", [])
