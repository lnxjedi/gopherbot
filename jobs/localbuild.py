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

# localbuild and all build types always get a "command" from gopherci,
# never triggered.
# One of "build", "depbuild", "job"
command = sys.argv.pop(0)

repository = sys.argv.pop(0)
branch = sys.argv.pop(0)

bot.SetParameter("GOPHERCI_BRANCH", branch)
pipeline = "pipeline"

if command == "job":
    pipeline = sys.argv.pop(0)
    bot.SetParameter("GOPHERCI_CUSTOM_PIPELINE", pipeline)

if command == "depbuild":
    deprepo = sys.argv.pop(0)
    depbranch = sys.argv.pop(0)
    bot.SetParameter("GOPHERCI_DEPBUILD", "true")
    bot.SetParameter("GOPHERCI_DEPREPO", deprepo)
    bot.SetParameter("GOPHERCI_DEPBRANCH", depbranch)

repodata = bot.GetRepoData()

if repository in repodata:
    repoconf = repodata[repository]

clone_url = repoconf["CloneURL"]
if len(clone_url) == 0:
    bot.Say("No 'clone_url' specified for '%s' in repositories.yaml" % repository)
    exit(1)

keep_history = repoconf["KeepLogs"]
if keep_history == None:
    keep_history = -1

# Protect the repository directory with Exclusive
if not bot.Exclusive(repository, False):
    bot.Log("Warn", "Build of '%s' already in progress, exiting" % repository)
    if len(bot.user) > 0:
        bot.Say("build for '%s' already in progress, not starting a new build for branch '%s'" % (repository, branch))
    exit()

repobranch = "%s/%s" % (repository, branch)
bot.ExtendNamespace(repobranch, keep_history)

bot.AddTask("start-build", [])
bot.AddTask("git-init", [ clone_url ])
# Start with a clean jobdir
bot.AddTask("cleanup", [ repository ])
bot.AddTask("git-clone", [ clone_url, branch, repository, "true" ])
bot.AddTask("run-pipeline", [ pipeline ] + sys.argv)
bot.FinalTask("finish-build", [])
