#!/bin/bash -e

# tasks/notify.sh - send notification to a user, generally used as a FailTask
# Requires two arguments: notify user and message

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

USER=$1
MESSAGE=$2

if [ -n "$GOPHER_CHANNEL" ]
then
    SendUserChannelMessage "$USER" "$GOPHER_CHANNEL" "$MESSAGE"
else
    SendUserMessage "$USER" "$MESSAGE"
fi