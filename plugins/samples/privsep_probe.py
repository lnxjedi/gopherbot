#!/usr/bin/env python3
"""
privsep_probe.py - manual privilege-separation probe for external Python plugins.

Sample robot.yaml entries:

ExternalPlugins:
  "privsep-external-priv":
    Description: Manual privsep probe (external python, privileged)
    Path: plugins/samples/privsep_probe.py
    Privileged: true
  "privsep-external-unpriv":
    Description: Manual privsep probe (external python, unprivileged)
    Path: plugins/samples/privsep_probe.py
    Privileged: false

Sample conf/plugins overlays:
  conf/plugins/privsep-external-priv.yaml.sample
  conf/plugins/privsep-external-unpriv.yaml.sample

Usage:
  - Enable one or both plugin entries above
  - Copy the sample overlay(s) to .yaml, adjust channels if needed
  - Ensure script is executable: chmod 755 plugins/samples/privsep_probe.py
  - Run:
      ;privsep external priv
      ;privsep external unpriv
"""

import os
import sys
from gopherbot_v2 import Robot

DEFAULT_CONFIG = """---
Help:
- Keywords: [ "privsep", "external" ]
  Helptext:
  - "(bot), privsep external priv - run external privsep probe in #general"
  - "(bot), privsep external unpriv - run external privsep probe in #random"
CommandMatchers: []
"""

SENSITIVE_KEYS = [
    "GOPHER_ENCRYPTION_KEY",
    "GOPHER_DEPLOY_KEY",
    "GOPHER_HOST_KEYS",
]


def probe(bot):
    home = os.getenv("GOPHER_HOME", "")
    env_path = os.path.join(home, ".env") if home else ".env"
    env_result = "deny"
    env_error = ""
    try:
        with open(env_path, "r", encoding="utf-8") as f:
            first = f.readline()
        env_result = f"ok({len(first)}b)"
    except Exception as exc:
        env_result = "deny"
        env_error = str(exc)

    parts = [
        "probe=external-python",
        f"uid={os.getuid()}",
        f"euid={os.geteuid()}",
        f"channel={os.getenv('GOPHER_CHANNEL', '')}",
        f"user={os.getenv('GOPHER_USER', '')}",
        f"envread={env_result}",
    ]
    if env_error:
        parts.append(f"error={env_error}")

    for key in SENSITIVE_KEYS:
        parts.append(f"{key}={'set' if os.getenv(key) else 'unset'}")

    bot.SayThread("PRIVSEP_PROBE_RESULT " + " | ".join(parts))


command = sys.argv[1] if len(sys.argv) > 1 else ""
if command == "configure":
    print(DEFAULT_CONFIG)
    sys.exit(0)

bot = Robot()
if command == "probe":
    probe(bot)

sys.exit(0)
