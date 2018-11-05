#!/bin/bash -e

# tasks/dmnotify.sh - send DM to a user, generally used as a FailTask
# Requires two arguments: notify user and message

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ $# -ne 2 ]
then
    Log "Error" "dmnotify called with num args != 2"
    exit 0
fi

USER=$1
MESSAGE=$2
SendUserMessage "$USER" "$MESSAGE"
