#!/bin/bash -e

# backup.sh - back up job for backing up the robot's state (brain)
# Note: significant changes here should probably be done to save.sh, too

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

if ! Exclusive "backup"
then
    Log "Info" "Unable to get exclusive access to 'backup', exiting"
    exit 0
fi

if [ "$GOPHER_STATE_REPOSITORY" ]
then
    PUSHBRANCH="${GOPHER_STATE_BRANCH:-master}"
    cd "$GOPHER_STATEDIR"
    if [ ! -d .git ]
    then
        NEWREPO="true"
        git init
        git remote add origin $GOPHER_STATE_REPOSITORY
        FailTask exec rm -rf ".git"
    fi
else
    GOPHER_STATE_REPOSITORY="$GOPHER_CUSTOM_REPOSITORY"
    PUSHBRANCH="${GOPHER_STATE_BRANCH:-robot-state}"
    if [ ! -d "$GOPHER_STATEDIR/.git" ]
    then
        if [ ! -d "$GOPHER_CONFIGDIR/.git" ]
        then
            Log "Error" "Backup to state branch specified with $GOPHER_CONFIGDIR/.robot-state, but $GOPHER_CONFIGDIR/.git doesn't exist"
            exit 1
        fi
        if [ -d "$GOPHER_STATEDIR/custom" ]
        then
            Log "Error" "$GOPHER_STATEDIR/custom already exists during initialization of backup"
            exit 1
        fi
        NEWREPO="true"
        # NOTE: technically, with no exclusive lock, GOPHER_CONFIGDIR
        # could change during the copy; however, this only happens once
        # on the first backup.
        cp -a "$GOPHER_CONFIGDIR" "$GOPHER_STATEDIR/custom"
        cd "$GOPHER_STATEDIR/custom"
        git checkout --orphan robot-state
        git rm -rf .
        mv .git/ ..
        cd ..
        rm -rf custom/
        CONFIGREPO=$(git remote get-url origin)
        GOPHER_STATE_REPOSITORY="$CONFIGREPO"
        FailTask exec rm -rf ".git"
    else
        cd "$GOPHER_STATEDIR"
    fi
fi

if [ -e ".failed" ]
then
    rm ".failed"
    FAILED="true"
fi
CHANGES=$(git status --porcelain)

if [ ! "$CHANGES" -a ! "$NEWREPO" -a ! "$FAILED" ] # no changes
then
    if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
    then
        Say "No changes, exiting..."
    fi
    exit 0
fi

SetWorkingDirectory "$GOPHER_STATEDIR"
if [ "$NEWREPO" ]
then
    # Default gitignore, don't back up histories
    echo 'bot:histories*' > .gitignore
    AddTask git-init "$GOPHER_STATE_REPOSITORY"
else
    ORIGIN=$(git remote get-url origin)
    AddTask git-init "$ORIGIN"
    FailTask exec touch ".failed"
fi
if [ "$CHANGES" ]
then
    AddTask pause-brain
    AddTask exec git add --all
    AddTask resume-brain
    AddTask exec git commit -m "Automated backup of robot state"
fi
if [ "$NEWREPO" ]
then
    AddTask exec git push -u origin $PUSHBRANCH
else
    AddTask exec git push
fi
if [ "$PTYPE" == "plugCommand" -o "$PTYPE" == "jobCommand" ]
then
    AddTask say "Backup complete"
fi
