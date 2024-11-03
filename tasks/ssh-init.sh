#!/bin/bash -e

# ssh-init.sh - pipeline task for setting up SSH options for the robot.
# This script now focuses on setting SSH_OPTIONS and GIT_SSH_COMMAND,
# assuming that the SSH agent is already running (handled by ssh-agent task).

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -z "$GOPHER_CONFIGDIR" ]; then
    MESSAGE="GOPHER_CONFIGDIR not set"
    Log "Error" "$MESSAGE"
    echo "$MESSAGE" >&2
    exit 1
fi

if [ -z "$SSH_AUTH_SOCK" ]; then
    Log "Error" "SSH_AUTH_SOCK not set; SSH agent task must have failed"
    exit 1
else
    Log "Debug" "ssh-init proceeding; SSH agent is running"
fi

SSH_OPTIONS="-o PasswordAuthentication=no"

if [ -z "$GOPHERBOT_IDE" ] && [ -n "$GOPHER_HOME" ]; then
    if [ -e "$GOPHER_CONFIGDIR/ssh/config" ]; then
        chmod 0600 "$GOPHER_CONFIGDIR/ssh/config"
        SSH_OPTIONS="$SSH_OPTIONS -F $GOPHER_CONFIGDIR/ssh/config"
    fi
    SSH_OPTIONS="$SSH_OPTIONS -o UserKnownHostsFile=$GOPHER_HOME/known_hosts"
fi

SetParameter SSH_OPTIONS "$SSH_OPTIONS"
SetParameter GIT_SSH_COMMAND "ssh $SSH_OPTIONS"

exit 0
