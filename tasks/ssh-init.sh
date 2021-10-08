#!/bin/bash -e

# ssh-init.sh - pipeline task for setting up an ssh-agent for the robot.
# See jobs/ssh-job.sh for info on generic ssh jobs and tasks.

# NOTE:
# The use of a FIFO and cat <<EOF is to prevent the passphrase from ever
# being stored in a file or appearing in the process list; otherwise this
# script might be a bit shorter.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -z "$GOPHER_CONFIGDIR" ]
then
    MESSAGE="GOPHER_CONFIGDIR not set"
    Log "Error" "$MESSAGE"
    echo "$MESSAGE" >&2
    exit 1
fi

if [ "$SSH_AGENT_PID" ]
then
    Log "Debug" "ssh-init exiting; ssh-agent already running"
    exit 0 # already running
fi

if [ "$BOOTSTRAP" ]
then
    Log "Info" "ssh-init starting in bootstrap mode"
elif [ "$GOPHERBOT_IDE" ]
then
    Log "Info" "ssh-init starting in Gopherbot IDE mode (without loading a key)"
elif [ ! "$KEYNAME" ]
then
    MESSAGE="KEYNAME not set"
    Log "Error" "$MESSAGE"
    echo "$MESSAGE" >&2
    exit 1
else
    SSH_KEY_PATH="$GOPHER_CONFIGDIR/ssh/$KEYNAME"
    if [ ! -e $SSH_KEY_PATH ]
    then
        Log "Error" "ssh/$KEYNAME not found in ssh-init, exiting"
        exit 1
    fi

    if [ -z "$BOT_SSH_PHRASE" ]
    then
        Log "Error" "I don't know the passphrase for my ssh keypair, aborting"
        exit 1
    fi
    chmod 600 "$SSH_KEY_PATH"
fi

export SSH_ASKPASS=$GOPHER_INSTALLDIR/helpers/ssh-askpass.sh
export SSH_ASKPASS_REQUIRE=force
export DISPLAY=""

eval `ssh-agent`

# Add cleanup task
FinalTask exec ssh-agent -k

if [ "$BOOTSTRAP" ]
then
    if [ -z "$GOPHER_DEPLOY_KEY" ]
    then
        Log "Error" "Bootstrap given but GOPHER_DEPLOY_KEY unset"
        exit 1
    fi
    echo "$GOPHER_DEPLOY_KEY" | tr '_:' ' \n' | ssh-add -
elif [ ! "$GOPHERBOT_IDE" ]
then
    ssh-add $SSH_KEY_PATH < /dev/null
fi

# Make agent available to other tasks in the pipeline
SetParameter SSH_AUTH_SOCK $SSH_AUTH_SOCK
SetParameter SSH_AGENT_PID $SSH_AGENT_PID

SSH_OPTIONS="-o PasswordAuthentication=no"
if [ ! "$GOPHERBOT_IDE" -a "$GOPHER_HOME" ]
then
    if [ -e "$GOPHER_CONFIGDIR/ssh/config" ]
    then
        chmod 0600 "$GOPHER_CONFIGDIR/ssh/config"
        SSH_OPTIONS="$SSH_OPTIONS -F $GOPHER_CONFIGDIR/ssh/config"
    fi
    SSH_OPTIONS="$SSH_OPTIONS -o UserKnownHostsFile=$GOPHER_HOME/known_hosts"
fi

SetParameter SSH_OPTIONS "$SSH_OPTIONS"
SetParameter GIT_SSH_COMMAND "ssh $SSH_OPTIONS"

exit 0
