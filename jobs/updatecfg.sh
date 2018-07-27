#!/bin/bash

# updatecfg.sh - job to update the robot's configuration and reload, assuming
# $GOPHER_CONFIGDIR is a git repository. This also assumes use of an SSH
# deploy key; see ssh-init.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

SetWorkingDirectory $GOPHER_CONFIGDIR
AddTask ssh-init
AddTask git pull
AddTask builtInadmin reload