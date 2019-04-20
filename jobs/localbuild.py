#!/usr/bin/env python

# localbuild.py - Clone a repository locally and run .gopherci/pipeline.sh

# localbuild is one of possibly several build types for a repository. When
# called with two arguments, they are interpreted as the repository and branch
# of a primary build. When called with four arguments, the first two are the
# repository and branch to build, and the second two are the repository and
# branch that triggered the build.
#
# The build type is responsible for calling Exclusive, setting up the build
# directory, and adding the initial pipeline tasks. All other
# pipeline/dependency logic is in gopherci.

import os
import re
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v1 import Robot

bot = Robot()

# Pop off the executable path
sys.argv.pop(0)

repository = sys.argv.pop(0)
branch = sys.argv.pop(0)
bot.SetParameter("GOPHERCI_REPO", repository)
bot.SetParameter("GOPHERCI_BRANCH", branch)
if len(sys.argv) > 0:
    deprepo = sys.argv.pop(0)
    depbranch = sys.argv.pop(0)
    bot.SetParameter("GOPHERCI_DEPBUILD", "true")
    bot.SetParameter("GOPHERCI_DEPREPO", deprepo)
    bot.SetParameter("GOPHERCI_DEPBRANCH", depbranch)

repodata = bot.GetRepoData()

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

repobranch = "%s/%s" % (repository, branch)
if not bot.Exclusive(repobranch, False):
    bot.Log("Warn", "Build of '%s' already in progress, exiting" % repobranch)
    if len(bot.user) > 0:
        bot.Say("localbuild of '%s' already in progress, not starting a new build" % repobranch)
    exit()

bot.ExtendNamespace(repobranch, keep_history)

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

bot.AddTask("git-sync", [ clone_url, branch, repobranch, "true" ])
bot.AddTask("runpipeline", [])
# TODO: eventually allow flag to leave the repo on failed builds?
# NOTA BENE: final tasks are executed in reverse order; adding these
# now means cleanup will be the last FinalTask to run, immediately
# preceded by resetting the workdir.
bot.FinalTask("cleanup", [])
bot.FinalTask("setworkdir", [ "." ])
