#!/bin/bash

# save.sh - save robot's configuration to GOPHER_CUSTOM_REPOSITORY
# Note: significant changes here should probably be done to backup.sh, too

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

PTYPE="$GOPHER_PIPELINE_TYPE"

if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
then
    Say "Starting config save requested by user $GOPHER_USER in channel: $GOPHER_START_CHANNEL"
fi

if [ ! "$GOPHER_CUSTOM_REPOSITORY" ]
then
    Log "Error" "GOPHER_CUSTOM_REPOSITORY not set"
    exit 1
fi

if ! Exclusive "save"
then
    Log "Info" "Unable to get exclusive access to 'save', exiting"
    exit 0
fi

cd $GOPHER_CONFIGDIR
if [ ! -d .git ]
then
    NEWREPO="true"
    git init
    git remote add origin $GOPHER_CUSTOM_REPOSITORY
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

SetWorkingDirectory "$GOPHER_CONFIGDIR"
AddTask git-init "$GOPHER_CUSTOM_REPOSITORY"
AddTask exec git add --all
AddTask exec git commit -m "Save robot configuration"
if [ "$NEWREPO" ]
then
    AddTask exec git push -u origin master
    FailTask exec rm -rf .git
else
    AddTask exec git push
fi
if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
then
    AddTask say "Save finished"
fi
