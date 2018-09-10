#!/usr/bin/env python

# gopherci.py - Dispatcher for commit events, spawns the appropriate build job

import os
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v1 import Robot

bot = Robot()

from yaml import load

# Pop off the executable path
sys.argv.pop(0)

repository = sys.argv.pop(0)
branch = sys.argv.pop(0)
if branch.endswith("/"): # illegal end char; assume args swapped
    repository, branch = branch, repository

if repository.endswith("/"):
    repository = repository.rstrip("/")

repofile = open("%s/conf/repositories.yaml" % os.getenv("GOPHER_CONFIGDIR"))
yamldata = repofile.read()
repodata = load(yamldata)

if not isinstance(repodata, dict):
    bot.Log("Warn", "GopherCI triggered with invalid 'repositories.yaml', not a python 'dict'")
    exit(0)

spawned = False

def spawn(reponame, repoconf, spawntype):
    if "type" in repoconf:
        repotype = repoconf["type"]
        if repotype == "none":
            bot.Log("Debug", "Ignoring update on %s repository '%s', type is 'none'" % (spawntype, repository))
        else:
            bot.Log("Debug", "gopherci spawning build for %s repository '%s', type '%s'" % (spawntype, repository, repotype))
            bot.SpawnTask(repotype, [ reponame, branch ])
    else:
        bot.Say("No 'type' specified for %s repository '%s'" % (spawntype, repository))

if repository in repodata:
    spawned = True
    spawn(repository, repodata[repository], "listed")

for reponame in repodata.keys():
    if "dependencies" in repodata[reponame]:
        if repository in repodata[reponame]["dependencies"]:
            spawned = True
            spawn(reponame, repodata[reponame], "dependency")

if not spawned:
    bot.Log("Debug", "Ignoring update on '%s', not a listed repository or dependency" % repository)
