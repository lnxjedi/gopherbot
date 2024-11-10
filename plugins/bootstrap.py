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
if not clone_url:
    bot.Log("Warn", "GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping")
    exit(0)

clone_branch = os.getenv("GOPHER_CUSTOM_BRANCH")

depkey = os.getenv("GOPHER_DEPLOY_KEY")
if not depkey:
    bot.Log("Error", "SSH required for bootstrapping and no GOPHER_DEPLOY_KEY set")
    exit(1)

bot.Log("Info", f"Creating bootstrap pipeline for {clone_url}")
bot.SetParameter("BOOTSTRAP", "true")
bot.SetParameter("GOPHER_DEPLOY_KEY", depkey)

# Ensure that ssh-agent is running with the deployment key
bot.AddTask("ssh-agent", ["deploy"])

# Handle host key verification
host_keys = os.getenv("GOPHER_HOST_KEYS")
insecure_clone = os.getenv("GOPHER_INSECURE_CLONE") == "true"

if host_keys:
    bot.AddTask("ssh-git-helper", ["addhostkeys", host_keys])
else:
    bot.SetParameter("GOPHER_INSECURE_CLONE", "true" if insecure_clone else "false")
    bot.AddTask("ssh-git-helper", ["loadhostkeys", clone_url])

# Remove ssh-init and git-init tasks as they are no longer needed
# bot.AddTask("ssh-init", [])
# bot.AddTask("git-init", [clone_url])

# Remove any temporary deployment keys
tmp_key_name = "binary-encrypted-key"
deploy_env = os.getenv("GOPHER_ENVIRONMENT")
if deploy_env != "production":
    tmp_key_name += "." + deploy_env
tkey = os.path.join(cfgdir, tmp_key_name)
bot.AddTask("exec", ["rm", "-f", tkey])

# Touch .restore to trigger restore if needed
bot.AddTask("exec", ["touch", ".restore"])

# Use the new git-command task with the clone subcommand
bot.AddTask("git-command", ["clone", clone_url, clone_branch, cfgdir])

# Restart the robot to apply changes
bot.AddTask("restart-robot", [])