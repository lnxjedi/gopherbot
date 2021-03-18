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

def error_finish():
    log_ref = os.getenv("GOPHER_LOG_REF")
    bot.Say("There were errors polling github repositories, log: %s" % log_ref)
    exit(0)

token = os.getenv("GITHUB_TOKEN", '...')
if token == '...':
    bot.Log("Error", "GITHUB_TOKEN not found in environment")
    error_finish()

repodata = bot.GetRepoData()

if not isinstance(repodata, dict):
    bot.Log("Warn", "github-poller triggered with invalid 'repositories.yaml', not a python 'dict'")
    error_finish()

prefix = "https://api.github.com/repos"
fetch_error = False
remotes = {}
want_builds = {}

def fetch_refs(url):
    global fetch_error
    req = urllib.request.Request(url=url)
    req.add_header('Authorization', f'token {token}')
    try:
        res = urllib.request.urlopen(req)
    except urllib.error.HTTPError as e:
        fetch_error = True
        bot.Log("Error", "Fetching '%s': (%d) %s" % (url, e.code, e.read()))
        return {}, False
    body = res.read()
    return json.loads(body.decode("utf-8")), True

def read_repo(reponame, repoconf):
    _, org, name = reponame.split("/")
    fullname = "%s/%s" % (org, name)
    print("Checking repo %s" % reponame)
    refs = {}

    tagurl = "%s/%s/tags" % (prefix, fullname)
    tags, ok = fetch_refs(tagurl)
    if ok:
        for t in list(tags):
            name = t["name"]
            commit = t["commit"]["sha"]
            print("Found %s : %s / %s" % (reponame, name, commit))
            refs[name] = commit
    else:
        return {}

    branchurl = "%s/%s/branches" % (prefix, fullname)
    branches, ok = fetch_refs(branchurl)
    if ok:
        for b in list(branches):
            name = b["name"]
            commit = b["commit"]["sha"]
            print("Found %s : %s / %s" % (reponame, name, commit))
            refs[name] = commit
    else:
        return {}

    return refs

# Read refs for all github repos and store in remotes dict
for reponame in repodata.keys():
    host, org, name = reponame.split("/")
    if host == "github.com":
        repoconf = repodata[reponame]
        repotype = repoconf["Type"]
        if len(repotype) != 0 and repotype != "none":
            refs = read_repo(reponame, repoconf)
            if len(refs) > 0:
                remotes[reponame] = refs

# Retrive repo status memory
memory = bot.CheckoutDatum("repostats", True)
if not memory.exists:
    memory.datum = {}
repostats = memory.datum

want_builds = {}

for reponame in remotes.keys():
    repostat = {}
    first_seen = False
    if not reponame in repostats:
        first_seen = True
        repostats[reponame] = {}
    repostat = repostats[reponame]
    refs = remotes[reponame]
    for name in list(refs):
        commit = refs[name]
        last = ""
        if name in repostat:
            last = repostat[name]
        repostat[name] = commit
        changed = False
        if commit != last:
            changed = True
        print("Evaluating %s / %s: last built: %s, current: %s, changed: %s" % (reponame, name, last, commit, changed))
        if changed:
            repotype = repoconf["Type"]
            if first_seen:
                print("Skipping primary build for %s (branch %s) to the pipeline, type '%s' (first time seen)" % (reponame, name, repotype))
            else:
                print("Adding primary build for %s (branch %s) to the pipeline, type '%s'" % (reponame, name, repotype))
                want_builds[reponame] = name
    for name in list(repostat):
        if not name in refs:
            print("Pruning %s from %s, no longer present" % (name, reponame))
            repostat.pop(name)

memory.datum = repostats
ret = bot.UpdateDatum(memory)
if ret != Robot.Ok:
    bot.Log("Error", "Unable to save long-term memory in github-poller: %s" % ret)
    error_finish()

for reponame in list(want_builds):
    name = want_builds[reponame]
    bot.SpawnJob("gopherci", [ "build", reponame, name ])

if fetch_error:
    error_finish()
