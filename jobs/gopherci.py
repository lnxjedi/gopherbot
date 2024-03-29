#!/usr/bin/env python3

# gopherci.py - Dispatcher for commit events, spawns the appropriate build job.
# NOTE: Don't use SetParameter(...) here; build_triggered jobs don't inherit
# environment.

# Operation:
# When a repository updates, gopherci is called with the repository
# name and branch (two arguments).
# - If the repository is listed in "repositories.yaml" with
#   type != none, a build task is added.
# - The the repository is listed as a dependency for another repository whose
#   type != none, another gopherci task is added with a third argument of "true"
# When the gopherci job runs with three arguments, each dependendent build
# is spawned.
#
# The result is if the initial build succeeds, all dependent builds will run
# in parallel with no further interdependencies.
#
# NOTE: current gopherci does not cascade dependent builds; if a dependency
# build is itself a de

import sys
from gopherbot_v2 import Robot

bot = Robot()

repodata = bot.GetRepoData()

if not isinstance(repodata, dict):
    bot.Log("Warn", "GopherCI triggered with invalid 'repositories.yaml', not a python 'dict'")
    exit(0)

build_triggered = False

# Pop off the executable path
sys.argv.pop(0)

def get_deps(repository, recurse, all_deps = []):
    deps = []
    for reponame in repodata.keys():
        if repodata[reponame]["Dependencies"] != None:
            if repository in repodata[reponame]["Dependencies"]:
                repoconf = repodata[reponame]
                repotype = repoconf["Type"]
                if len(repotype) != 0 and repotype != "none":
                    if reponame in all_deps:
                        raise Exception("Found duplicate dependency %s while walking the dependency tree" % reponame)
                    deps.append(reponame)
                    all_deps.append(reponame)
    if recurse:
        if not deps:
            return deps
        for dep in deps:
            get_deps(dep, True, all_deps)

    return deps

if len(sys.argv) == 2:
    command = "build"
else:
    command = sys.argv.pop(0)

repository = sys.argv.pop(0)
branch = sys.argv.pop(0)

if command == "build":
    if branch.endswith("/"): # illegal end char; assume args swapped
        repository, branch = branch, repository

    if repository.endswith("/"):
        repository = repository.rstrip("/")
    if repository in repodata:
        repoconf = repodata[repository]
        if "Type" in repoconf:
            repotype = repoconf["Type"]
            if repotype != "none":
                build_triggered = True
                bot.Log("Debug", "Adding primary build for %s / %s to the pipeline" % (repository, branch))
                bot.AddJob(repotype, [ "build", repository, branch ])
    try:
        deps = get_deps(repository, True)
    except Exception as e:
        err = "Resolving dependencies for %s / %s: %s" % (repository, branch, e)
        bot.Log("Error", err)
        bot.AddTask("status", [ err ])
        exit(0)
    if deps:
        build_triggered = True
        bot.Log("Debug", "Starting builds for everything that depends on %s / %s" % (repository, branch))
        bot.AddJob("gopherci", [ "builddeps", repository, branch ])

if command == "job":
    # Run a custom pipeline
    pipeline = sys.argv.pop(0)
    if repository in repodata:
        repoconf = repodata[repository]
        if "Type" in repoconf:
            repotype = repoconf["Type"]
            if repotype != "none":
                bot.Log("Debug", "Adding custom job for %s / %s to the pipeline, running pipeline: %s" % (repository, branch, pipeline))
                bot.AddJob(repotype, [ "job", repository, branch, pipeline ] + sys.argv )
                exit()
    bot.Log("Error", "Missing repository '%s' or repository Type for custom job: %s" % (repository, pipeline))
    exit()

if command == "builddeps":
    # build depdencies for a repository
    build_triggered = True
    for reponame in repodata.keys():
        if repodata[reponame]["Dependencies"] != None:
            if repository in repodata[reponame]["Dependencies"]:
                repoconf = repodata[reponame]
                repotype = repoconf["Type"]
                if len(repotype) != 0 and repotype != "none":
                    if "default_branch" in repoconf:
                        repobranch = repoconf["default_branch"]
                    else:
                        repobranch = "."
                    bot.Log("Debug", "Spawning dependency build of %s / %s for primary build of %s / %s" % (reponame, repobranch, repository, branch))
                    bot.SpawnJob("gopherci", [ "depbuild", reponame, repobranch, repository, branch ])

if command == "depbuild":
    # Inital build of dependency
    # build repo + deps
    build_triggered = True
    deprepo = sys.argv.pop(0)
    depbranch = sys.argv.pop(0)
    if repository in repodata:
        repoconf = repodata[repository]
        repotype = repoconf["Type"]
        if len(repotype) != 0 and repotype != "none":
            build_triggered = True
            bot.Log("Debug", "Adding primary dependency build for %s / %s to the pipeline, triggered by %s / %s" % (repository, branch, deprepo, depbranch))
            bot.AddJob(repotype, [ "depbuild", repository, branch, deprepo, depbranch ])
    deps = get_deps(repository, False)
    if deps:
        bot.Log("Debug", "Starting builds for everything that depends on %s / %s (initially triggered by %s / %s" % (repository, branch, deprepo, depbranch))
        bot.AddJob("gopherci", [ "builddeps", repository, branch ])

if not build_triggered:
    bot.Log("Debug", "Ignoring update on '%s', no builds triggered" % repository)
