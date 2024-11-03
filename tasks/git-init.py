#!/usr/bin/env python3

# git-init.py - task for setting up git credentials

import os
import re
import sys
from gopherbot_v2 import Robot

bot = Robot()

cfgdir = os.getenv("GOPHER_CONFIGDIR")
try:
    os.stat(f"{cfgdir}/git/config")
except FileNotFoundError:
    bot.Log("Warn", f"{cfgdir}/git/config not found, git push will fail")

bot.SetParameter("XDG_CONFIG_HOME", cfgdir)

# Pop off the executable path
sys.argv.pop(0)

if len(sys.argv) < 1:
    bot.Log("Error", "No clone URL provided")
    sys.exit(1)

clone_url = sys.argv.pop(0)

if clone_url.startswith("http"):
    bot.Log("Error", f"HTTP(s) clone URLs are not deprecated: {clone_url}")
    sys.exit(1)

# Since we now assume that the clone URL is always an SSH URL, proceed accordingly
# Extract the host from the SSH clone URL
match = re.match(r"ssh://(?:.*@)?([^:/]*)(?::(\d+))?", clone_url)
if match:
    scanhost = match.group(1)
    port = match.group(2)
    if port:
        scanhost = f"{scanhost}:{port}"
else:
    # Handle scp-like syntax: [user@]host:path
    match = re.match(r"(?:.*@)?([^:/]*)[:/].*", clone_url)
    if match:
        scanhost = match.group(1)
    else:
        bot.Log("Error", f"Failed to parse SSH clone URL: {clone_url}")
        sys.exit(1)

bot.AddTask("ssh-scan", [scanhost])
