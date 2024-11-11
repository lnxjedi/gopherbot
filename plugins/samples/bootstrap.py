#!/usr/bin/env python3

# bootstrap.py - bootstrap a robot from env vars

"""
DEPRECATED bootstrap.py - this was the last version of the python-based bootstrapping process
for robots. It has since been replaced by 'gojobs/go-bootstrap/go_bootstrap_job.go'.

This example is kept here because it illustrates the use of the 'ssh-agent', 'ssh-git-helper'
and 'git-clone' tasks.
"""

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

# Add the ssh-agent task before git operations
bot.AddTask("ssh-agent", ["deploy"])

# Add ssh-git-helper task to handle host key verification
host_keys = os.getenv("GOPHER_HOST_KEYS")
insecure_clone = os.getenv("GOPHER_INSECURE_CLONE") == "true"

if host_keys:
    bot.AddTask("ssh-git-helper", ["addhostkeys", host_keys])
else:
    bot.SetParameter("GOPHER_INSECURE_CLONE", "true" if insecure_clone else "false")
    bot.AddTask("ssh-git-helper", ["loadhostkeys", clone_url])

# Set SSH_OPTIONS and GIT_SSH_COMMAND 
bot.AddTask("ssh-git-helper", ["publishenv"])

tmp_key_name = "binary-encrypted-key"
deploy_env = os.getenv("GOPHER_ENVIRONMENT")
if deploy_env != "production":
    tmp_key_name += "." + deploy_env
tkey = os.path.join(cfgdir, tmp_key_name)
bot.AddTask("exec", ["rm", "-f", tkey])
# touch restore even if GOPHER_BRAIN != file;
# backup and restore will check and exit
bot.AddTask("exec", ["touch", ".restore"])
bot.AddTask("git-clone", [clone_url, clone_branch, cfgdir, "true"])
bot.AddTask("restart-robot", [])
