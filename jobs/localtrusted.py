#!/usr/bin/env python

# localtrusted.py - Clone a repository locally and run .gopherci/pipeline.sh

import os
import re
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v1 import Robot

bot = Robot()

from yaml import load

# Pop off the executable path
sys.argv.pop(0)

repository = sys.argv.pop(0)
branch = sys.argv.pop(0)

repofile = open("%s/conf/repositories.yaml" % os.getenv("GOPHER_CONFIGDIR"))
yamldata = repofile.read()

repodata = load(yamldata)

if repository in repodata:
    repoconf = repodata[repository]

if "clone_url" not in repoconf:
    bot.Say("No 'clone_url' specified for '%s' in repositories.yaml" % repository)
    exit()
clone_url = repoconf["clone_url"]

if "keep_history" not in repoconf:
    keep_history = 7
else:
    keep_history = repoconf["keep_history"]

if not bot.Exclusive(repository, False):
    bot.Log("Warn", "Build of '%s' already in progress, exiting" % repository)
    exit()

bot.ExtendNamespace(repository, keep_history)
match = re.match(r"ssh://(?:.*@)?([^:/]*)(?::([^/]*)/)?", clone_url)
if match:
    bot.AddTask("ssh-init", [])
    if match.group(2):
        bot.AddTask("ssh-scan", [ "-p", match.group(2), match.group(1) ])
    else:
        bot.AddTask("ssh-scan", [ match.group(1) ])
else:
    match = re.match(r"(?:.*@)?([^:/]*)", clone_url)
    if match:
        bot.AddTask("ssh-init", [])
        bot.AddTask("ssh-scan", [ match.group(1) ])
bot.SetParameter("GOPHERCI_REPO", repository)
bot.SetParameter("GOPHERCI_BRANCH", branch)
bot.AddTask("git-sync", [ clone_url, branch, repository, "true" ])
bot.AddTask("exec", [ ".gopherci/pipeline.sh" ])
