#!/bin/bash -e

# restore.sh - restore the robot's state from git

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}"
    # The script should usually exit on error
    exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

PTYPE=$GOPHER_PIPELINE_TYPE
if [ \( "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" \) -a "$GOPHER_USER" ]
then
    INTERACTIVE="true"
fi

report(){
    local LEVEL=$1
    local MESSAGE=$2
    Log "$LEVEL" "$MESSAGE"
    if [ "$INTERACTIVE" ]
    then
        Say "$MESSAGE"
    fi
}

# GOPHER_STATEDIR should be defined in the "manage" namespace
if [ -z "$GOPHER_STATEDIR" ]
then
    report "Error" "GOPHER_STATEDIR not defined, giving up"
    exit 0
fi

if [ -e "$GOPHER_STATEDIR/.git" -a "$1" != "force" ]
then
    report "Warn" "'$GOPHER_STATEDIR/.git' exists, use 'force' to restore anyway"
    exit 0
fi

if [ ! "$GOPHER_STATE_REPOSITORY" ]
then
    CONFIGREPO=$(cd $GOPHER_CONFIGDIR; git remote get-url origin)
    GOPHER_STATE_REPOSITORY="$CONFIGREPO"
    GOPHER_STATE_BRANCH="${GOPHER_STATE_BRANCH:-robot-state}"
else
    GOPHER_STATE_BRANCH="${GOPHER_STATE_BRANCH:-master}"
fi

if ! Exclusive "backup"
then
    report "Warn" "Unable to get exclusive access to 'backup' in restore job, exiting"
    exit 0
fi

if [ "$INTERACTIVE" ]
then
    Say "Starting state restore requested by user $GOPHER_USER in channel: $GOPHER_START_CHANNEL"
fi

AddTask git-init "$GOPHER_STATE_REPOSITORY"
# Not certain this will all happen within lockMax, but *shrug*
AddTask pause-brain
AddTask cleanup "$GOPHER_STATEDIR"
AddTask git-sync "$GOPHER_STATE_REPOSITORY" "$GOPHER_STATE_BRANCH" "$GOPHER_STATEDIR"
AddTask resume-brain
if [ "$INTERACTIVE" ]
then
    AddTask say "Restore finished"
fi
