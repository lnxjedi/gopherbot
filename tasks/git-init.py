#!/usr/bin/env python3

# git-init.py - task for setting up git credentials
# Currently this only adds ssh-init and ssh-scan <host> to
# the pipeline if needed, but may eventually handle http credentials
# with a git credential helper.

# Usage: AddTask git-init <clone_url>

import os
import re
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v2 import Robot

bot = Robot()

cfgdir = os.getenv("GOPHER_CONFIGDIR")
try:
    os.stat("%s/git/config" % cfgdir)
except FileNotFoundError:
    bot.Log("Warn", "%s/git/config not found, git push will fail" % cfgdir)

bot.SetParameter("XDG_CONFIG_HOME", cfgdir)

# Pop off the executable path
sys.argv.pop(0)

clone_url = sys.argv.pop(0)

if not clone_url.startswith("http"):
    match = re.match(r"ssh://(?:.*@)?([^:/]*)(?::([^/]*)/)?", clone_url)
    if match:
        scanhost = match.group(1)
        if match.group(2):
            scanhost = "%s:%s" % ( scanhost, match.group(2) )
    else:
        match = re.match(r"(?:.*@)?([^:/]*)", clone_url)
        if match:
            scanhost = match.group(1)
    bot.AddTask("ssh-init", [])
    bot.AddTask("ssh-scan", [ scanhost ])
