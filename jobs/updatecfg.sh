#!/bin/bash

# updatecfg.sh - job to update or retrieve the robot's configuration and
# reload. Initializes ssh if ... TODO: rewrite me in python and be smart about ssh-init;
# see localtrusted.py

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

SetWorkingDirectory $GOPHER_CONFIGDIR
AddTask ssh-init
AddTask git pull
AddTask builtInadmin reload
