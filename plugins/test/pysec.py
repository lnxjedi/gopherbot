#!/usr/bin/python3

import sys
from gopherbot_v2 import Robot

default_config = """
---
AllChannels: true
AllowDirect: true
Authorizer: groups
AuthRequire: Helpdesk
AdminCommands:
- secadmincmd
AuthorizedCommands:
- secauthz
ElevatedCommands:
- secelevated
ElevateImmediateCommands:
- secimmediate
AllowedHiddenCommands:
- sechiddenok
CommandMatchers:
- Regex: (?i:python-sec-open)
  Command: secopen
- Regex: (?i:python-sec-admincmd)
  Command: secadmincmd
- Regex: (?i:python-sec-authz)
  Command: secauthz
- Regex: (?i:python-sec-elevated)
  Command: secelevated
- Regex: (?i:python-sec-immediate)
  Command: secimmediate
- Regex: (?i:python-sec-hidden-ok)
  Command: sechiddenok
- Regex: (?i:python-sec-hidden-denied)
  Command: sechiddendenied
"""


if len(sys.argv) < 2:
  sys.exit(1)

sys.argv.pop(0)
command = sys.argv.pop(0)

if command == "configure":
  print(default_config)
  sys.exit(0)

bot = Robot()

if command in [
  "secopen",
  "secadmincmd",
  "secauthz",
  "secauthall",
  "secelevated",
  "secimmediate",
  "sechiddenok",
  "sechiddendenied",
  "secadminonly",
  "secusersonly",
  "secmisconfig",
]:
  bot.Say("SECURITY CHECK: %s" % command)
  sys.exit(0)

sys.exit(1)
