#!/usr/bin/python3

import sys
from gopherbot_v2 import Robot

default_config = """
---
AllChannels: true
AllowDirect: true
"""


if len(sys.argv) < 2:
  sys.exit(1)

sys.argv.pop(0)
command = sys.argv.pop(0)

if command == "configure":
  print(default_config)
  sys.exit(0)
if command == "init":
  sys.exit(0)

bot = Robot()

if command == "encryptsecret":
  ciphertext, ret = bot.EncryptSecret("test-secret")
  if ret == Robot.Ok and ciphertext != "":
    bot.Say("ENCRYPT SECRET: ok")
  else:
    bot.Say("ENCRYPT SECRET: failed")
  sys.exit(0)

sys.exit(1)
