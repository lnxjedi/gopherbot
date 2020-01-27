#!/bin/bash

# backup.sh - back up job for backing up the robot's state (brain)
# Note: significant changes here should probably be done to save.sh, too

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

PTYPE="$GOPHER_PIPELINE_TYPE"

if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
then
    Say "Starting backup requested by user $GOPHER_USER in channel: $GOPHER_START_CHANNEL"
fi

if [ ! "$GOPHER_CUSTOM_REPOSITORY" ]
then
    Log "Error" "GOPHER_CUSTOM_REPOSITORY not set"
    exit 1
fi
DEFAULT_STATE_REPOSITORY=${GOPHER_CUSTOM_REPOSITORY/gopherbot/state}
GOPHER_STATE_REPOSITORY=${GOPHER_STATE_REPOSITORY:-$DEFAULT_STATE_REPOSITORY}

if ! Exclusive "backup"
then
    Log "Info" "Unable to get exclusive access to 'backup', exiting"
    exit 0
fi

cd $GOPHER_STATEDIR
if [ ! -d .git ]
then
    NEWREPO="true"
    git init
    git remote add origin $GOPHER_STATE_REPOSITORY
else
    CHANGES=$(git status --porcelain)
fi

if [ ! "$CHANGES" -a ! "$NEWREPO" ] # no changes
then
    if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
    then
        Say "No changes, exiting..."
    fi
    exit 0
fi

SetWorkingDirectory "$GOPHER_STATEDIR"
AddTask git-init "$GOPHER_STATE_REPOSITORY"
AddTask pause-brain
AddTask exec git add --all
AddTask resume-brain
AddTask exec git commit -m "Automated backup of robot state"
if [ "$NEWREPO" ]
then
    AddTask exec git push -u origin master
    FailTask exec rm -rf .git
else
    AddTask exec git push
fi
if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
then
    AddTask say "Backup complete"
fi
