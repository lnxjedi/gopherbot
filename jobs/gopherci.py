#!/usr/bin/env python

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

repofile = open("%s/conf/repositories.yaml" % os.getenv("GOPHER_CONFIGDIR"))
yamldata = repofile.read()

repodata = load(yamldata)

if repository in repodata:
    repoconf = repodata[repository]
    if "type" in repoconf:
        repotype = repoconf["type"]
    else:
        bot.Say("No 'type' specified for %s" % repository)
        exit()
else:
    bot.Log("Debug", "Ignoring update on '%s', not listed in repositories.yaml" % repository)
    exit()

bot.Say("Found '%s' in 'repositories.yaml', type '%s'" % (repository, repotype))
