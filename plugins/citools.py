#!/usr/bin/python3

import os
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v2 import Robot

bot = Robot()

executable = sys.argv.pop(0)
command = sys.argv.pop(0)

if command == "configure":
    exit(0)

repodata = bot.GetRepoData()

def start_build(repository, branch):
    bot.Say("Ok, I'll start the gopherci job for %s, %s branch..." % (repository, branch))
    bot.AddJob("gopherci", [ repository, branch ])
    bot.AddTask("say", ["... build completed"])
    bot.FailTask("say", ["... build failed"])

if not isinstance(repodata, dict):
    bot.Say("'repodata.yaml' missing or invalid; GetRepoData() failed")
    exit(0)

if command == "build":
    build = []
    repospec = sys.argv.pop(0).lower()
    fcount = len(repospec.split("/"))

    branch = sys.argv.pop(0)
    if len(branch) == 0:
        branch = "master"

    for reponame in repodata.keys():
        match = "/".join(reponame.lower().split("/")[-fcount:])
        if repospec == match:
            if reponame not in build:
                build.append(reponame)

    for reponame in repodata.keys():
        if "dependencies" in repodata[reponame]:
            for deprepo in repodata[reponame]["dependencies"]:
                match = "/".join(deprepo.lower().split("/")[-fcount:])
                if repospec == match:
                    if deprepo not in build:
                        build.append(deprepo)

    if len(build) == 0:
        bot.Say("I don't have any repositories matching %s" % repospec)
    elif len(build) > 1:
        bot.Say("Multiple repositories match %s, please qualify further" % repospec)
    else:
        start_build(build[0], branch)

if command == "help":
    bot.Say("""\
The "build <repo> (branch)" command takes a repository name and optional \
branch (default "master"), which the robot will try matching against \
repositories and dependencies in "repositories.yaml". When the repository \
name is not unique, you can also add a user/org component to match, e.g. \
"build joeblow/website"; that can be further qualified with the site name \
if needed, e.g. "build github.com/joeblow/website".""")
