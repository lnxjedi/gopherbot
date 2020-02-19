#!/bin/bash -e

# ssh-init.sh - pipeline task for setting up an ssh-agent for the robot. To
# use:
# - Copy conf/jobs/ssh-init.yaml.sample to ssh-init.yaml (as-is)
# - Add "ssh" and "ssh-agent" to ExternalTask list, with Name, Path and
#   Description (or uncomment)
# - Add job "ssh-init" to ExternalJobs (uncomment from sample) and reload
# - Set the ssh passphrase with a DM to the robot:
#   store task parameter ssh-init BOT_SSH_PHRASE=<bot ssh passphrase>
# - Put tasks in a pipeline, e.g.:
#    AddTask ssh-init
#    ... (do stuff)
#    AddTask ssh-agent -k

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

if [ -n "$SSH_AGENT_PID" ]
then
    Log "Debug" "ssh-init exiting; ssh-agent already running"
    exit 0 # already running
fi

if [ -n "$BOOTSTRAP" ]
then
    Log "Info" "ssh-init starting in bootstrap mode"
    BOOTSTRAP="true"
fi

SSH_KEY=${KEYNAME:-robot_rsa}
SSH_KEY_PATH="$GOPHER_CONFIGDIR/ssh/$SSH_KEY"

if [ -z "$BOOTSTRAP" ]
then
    if [ ! -e $SSH_KEY_PATH ]
    then
        Log "Warn" "No ssh key found in ssh-init, exiting"
        exit 0
    fi

    if [ -z "$BOT_SSH_PHRASE" ]
    then
        Log "Error" "I don't know the passphrase for my ssh keypair, aborting"
        exit 1
    fi
    chmod 600 "$SSH_KEY_PATH"
fi

export SSH_ASKPASS=$GOPHER_INSTALLDIR/helpers/ssh-askpass.sh
export DISPLAY=""

eval `ssh-agent`

# Add cleanup task
FinalTask exec ssh-agent -k

if [ -n "$BOOTSTRAP" ]
then
    if [ -z "$GOPHER_DEPLOY_KEY" ]
    then
        Log "Error" "Bootstrap given but GOPHER_DEPLOY_KEY unset"
        exit 1
    fi
    echo "$GOPHER_DEPLOY_KEY" | tr '_:' ' \n' | ssh-add -
else
    ssh-add $GOPHER_CONFIGDIR/ssh/$SSH_KEY < /dev/null
fi

# Make agent available to other tasks in the pipeline
SetParameter SSH_AUTH_SOCK $SSH_AUTH_SOCK
SetParameter SSH_AGENT_PID $SSH_AGENT_PID

SSH_OPTIONS="-o PasswordAuthentication=no"
if [ -n "$GOPHER_HOME" ]
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
