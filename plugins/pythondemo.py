#!/usr/bin/python

import os
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot import Robot

bot = Robot()
print "hello, world: %s" % bot.channel
