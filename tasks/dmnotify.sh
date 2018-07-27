#!/bin/bash -e

# tasks/dmnotify.sh - send DM to a user, generally used as a FailTask
# Requires two arguments: notify user and message

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

USER=$1
MESSAGE=$2
SendUserMessage "$USER" "$MESSAGE"