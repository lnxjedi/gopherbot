#!/bin/bash -e

# save.sh - save robot's configuration to GOPHER_CUSTOM_REPOSITORY
# Note: significant changes here should probably be done to backup.sh, too

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}" >&2
    # The script should usually exit on error
    exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

for REQUIRED in git jq ssh
do
    if ! which $REQUIRED >/dev/null 2>&1
    then
        echo "Required '$REQUIRED' not found in \$PATH"
        exit 1
    fi
done

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

FailTask tail-log

PTYPE="$GOPHER_PIPELINE_TYPE"

if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
then
    CHANNEL=${GOPHER_START_CHANNEL:-(direct message)}
    Say "Starting config save requested by user $GOPHER_USER in channel: $CHANNEL"
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
    git branch -m main
    git remote add origin $GOPHER_CUSTOM_REPOSITORY
else
    CHANGES=$(git status --porcelain)
    COMMITS=$(git cherry)
fi

if [ ! "$CHANGES" -a ! "$COMMITS" -a ! "$NEWREPO" ] # no changes
then
    if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
    then
        Say "No changes, exiting..."
    fi
    exit 0
fi

SetWorkingDirectory "$GOPHER_CONFIGDIR"
AddTask git-init "$GOPHER_CUSTOM_REPOSITORY" "-b" "main"
if [ "$CHANGES" -o "$NEWREPO" ]
then
    AddTask exec git add --all
    AddTask exec git commit -m "Save robot configuration"
fi
if [ "$NEWREPO" ]
then
    AddTask exec git push -u origin main
    FailTask exec rm -rf .git
else
    AddTask exec git push
fi
if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
then
    AddTask say "Save finished"
fi
