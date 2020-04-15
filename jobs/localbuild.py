#!/usr/bin/env python3

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
from gopherbot_v2 import Robot

bot = Robot()

# Pop off the executable path
sys.argv.pop(0)

repository = sys.argv.pop(0)
branch = sys.argv.pop(0)
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

if "CloneURL" not in repoconf:
    bot.Say("No 'clone_url' specified for '%s' in repositories.yaml" % repository)
    exit()
clone_url = repoconf["CloneURL"]

if "KeepHistory" not in repoconf:
    keep_history = 7
else:
    keep_history = repoconf["KeepHistory"]

repobranch = "%s/%s" % (repository, branch)
if not bot.Exclusive(repobranch, False):
    bot.Log("Warn", "Build of '%s' already in progress, exiting" % repobranch)
    if len(bot.user) > 0:
        bot.Say("localbuild of '%s' already in progress, not starting a new build" % repobranch)
    exit()

bot.ExtendNamespace(repobranch, keep_history)

bot.AddTask("start-build", [])
bot.AddTask("git-init", [ clone_url ])
# Start with a clean jobdir
bot.AddTask("cleanup", [ repobranch ])
bot.AddTask("git-clone", [ clone_url, branch, repobranch, "true" ])
bot.AddTask("run-pipeline", [])
bot.FinalTask("finish-build", [])
