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

def start_build(repository, branch, pipeline, args):
    if branch == ".":
        tell_branch = "(default)"
    else:
        tell_branch = branch
    if pipeline == "pipeline":
        bot.Say("Ok, I'll start the gopherci job for %s, %s branch..." % (repository, tell_branch))
        bot.AddJob("gopherci", [ "build", repository, branch ])
        bot.AddTask("say", ["... build of %s/%s completed" % (repository, tell_branch) ])
        bot.FailTask("say", ["... build of %s/%s failed" % (repository, tell_branch) ])
    else:
        bot.Say("Ok, I'll start the gopherci custom job for %s, %s branch, running pipeline: %s" % (repository, tell_branch, pipeline))
        bot.AddJob("gopherci", [ "job", repository, branch, pipeline ] + args)
        bot.AddTask("say", ["... job %s/%s - %s: completed" % (repository, tell_branch, pipeline) ])
        bot.FailTask("say", ["... job %s/%s - %s: failed" % (repository, tell_branch, pipeline) ])

if not isinstance(repodata, dict):
    bot.Say("'repodata.yaml' missing or invalid; GetRepoData() failed")
    exit(0)

if command == "build":
    build = []
    repospec = sys.argv.pop(0).lower()
    fcount = len(repospec.split("/"))

    branch = sys.argv.pop(0)
    if len(branch) == 0:
        branch = "."
    
    pipeline = sys.argv.pop(0)
    args = []
    if len(pipeline) == 0:
        pipeline = "pipeline"
    else:
        args = sys.argv.pop(0).split(" ")

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
        start_build(build[0], branch, pipeline, args)

if command == "help":
    bot.Say("""\
The "build <repo> (branch)" command takes a repository name and optional \
branch, which the robot will try matching against repositories and \
dependencies in "repositories.yaml". When the repository name is not \
unique, you can also add a user/org component to match, e.g. \
"build joeblow/website"; that can be further qualified with the site name \
if needed, e.g. "build github.com/joeblow/website".""")
