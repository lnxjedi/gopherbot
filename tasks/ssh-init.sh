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

if [ -z "$HOME" ]
then
    MESSAGE="HOME not set"
    Log "Error" "$MESSAGE"
    echo "$MESSAGE" >&2
    exit 1
fi

if [ -n "$SSH_AGENT_PID" ]
then
    Log "Debug" "ssh-init exiting; ssh-agent already running"
    exit 0 # already running
fi

SSH_KEY=${KEYNAME:-robot_rsa}
SSH_KEY_PATH="$HOME/.ssh/$SSH_KEY"
if [ ! -e $SSH_KEY_PATH ]
then
    Log "Warn" "No ssh key found in ssh-init, exiting"
    exit 0
fi

if grep -q ENCRYPTED $SSH_KEY_PATH
then
    if [ -z "$BOT_SSH_PHRASE" ]
    then
        MESSAGE="BOT_SSH_PHRASE not set, see conf/jobs/ssh-init.yaml"
        Log "Error" "$MESSAGE"
        echo "$MESSAGE" >&2
        exit 1
    fi
    export SSH_ASKPASS=$GOPHER_INSTALLDIR/scripts/ssh-askpass.sh
    export DISPLAY=""
fi

eval `ssh-agent`
# Add cleanup task
FinalTask ssh-agent -k

ssh-add $HOME/.ssh/$SSH_KEY < /dev/null

# Make agent available to other tasks in the pipeline
SetParameter SSH_AUTH_SOCK $SSH_AUTH_SOCK
SetParameter SSH_AGENT_PID $SSH_AGENT_PID

exit 0
