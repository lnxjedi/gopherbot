#!/usr/bin/env python3

# github-poller.py - Loop through repositories.yaml and check github repos
# to see if they've changed. If so, start a gopherci build job.
# GITHUB_TOKEN needs to be provided in `robot.yaml`

import os
import sys
import json
import urllib.request
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v2 import Robot

bot = Robot()
token = os.getenv("GITHUB_TOKEN", '...')
if token == '...':
    bot.Log("Error", "GITHUB_TOKEN not found in environment")
    exit(0)

repodata = bot.GetRepoData()

if not isinstance(repodata, dict):
    bot.Log("Warn", "github-poller triggered with invalid 'repositories.yaml', not a python 'dict'")
    exit(0)

# Pop off the executable path
sys.argv.pop(0)

prefix = "https://api.github.com/repos"
first_run = False

# Retrive repo status memory
memory = bot.CheckoutDatum("repostats", True)
if not memory.exists:
    first_run = True
    memory.datum = {}
repostats = memory.datum

want_builds = {}

def fetch_refs(url):
    req = urllib.request.Request(url=url)
    req.add_header('Authorization', f'token {token}')
    res = urllib.request.urlopen(req)
    body = res.read()
    return json.loads(body.decode("utf-8"))

def check_repo(reponame, repoconf):
    _, org, name = reponame.split("/")
    fullname = "%s/%s" % (org, name)
    if not fullname in repostats:
        repostats[fullname] = {}
    repostat = repostats[fullname]
    print("Checking repo %s" % fullname)
    refs = {}

    tagurl = "%s/%s/tags" % (prefix, fullname)
    tags = fetch_refs(tagurl)
    for t in list(tags):
        name = t["name"]
        commit = t["commit"]["sha"]
        refs[name] = commit

    branchurl = "%s/%s/branches" % (prefix, fullname)
    branches = fetch_refs(branchurl)
    for b in list(branches):
        name = b["name"]
        commit = b["commit"]["sha"]
        refs[name] = commit

    for name in list(refs):
        commit = refs[name]
        last = ""
        if name in repostat:
            last = repostat[name]
        repostat[name] = commit
        build = False
        if commit != last:
            build = True
        print("Found %s / %s: last built: %s, current: %s, build: %s" % (fullname, name, last, commit, build))
        if build:
            repotype = repoconf["Type"]
            if first_run:
                bot.Log("Debug", "Skipping primary build for %s (branch %s) to the pipeline, type '%s' (first run)" % (reponame, name, repotype))
            else:
                bot.Log("Debug", "Adding primary build for %s (branch %s) to the pipeline, type '%s'" % (reponame, name, repotype))
                want_builds[reponame] = name
    if len(refs) > 0:
        for name in list(repostat):
            if not name in refs:
                bot.Log("Debug", "Pruning %s from %s, no longer present" % (name, fullname))
                repostat.pop(name)

for reponame in repodata.keys():
    host, org, name = reponame.split("/")
    if host == "github.com":
        repoconf = repodata[reponame]
        repotype = repoconf["Type"]
        if len(repotype) != 0 and repotype != "none":
            check_repo(reponame, repoconf)

memory.datum = repostats
ret = bot.UpdateDatum(memory)
if ret != Robot.Ok:
    bot.Log("Error", "Unable to save long-term memory in github-poller: %s" % ret)
    exit(1)

for reponame in list(want_builds):
    name = want_builds[reponame]
    bot.SpawnJob("gopherci", [ "build", reponame, name ])
