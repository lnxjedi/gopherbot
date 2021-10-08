#!/bin/bash -e

# git-clone.sh - clone a git repository and optionally set the
# working directory

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ $# -lt 3 ]
then
    echo "Not enough arguments to git-clone; usage: git-clone <url> <branch> <dir> (true)" >&2
    exit 1
fi

REPO_URL=$1
BRANCH=$2
REPO_DIR=$3
SET_WD=$4

if [ -d "$REPO_DIR" -a "$(ls -A $REPO_DIR 2>/dev/null)" ]
then
    echo "Directory $REPO_DIR exists and is not empty" >&2
    exit 1
fi

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}" >&2
    # The script should usually exit on error
    exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

mkdir -p $REPO_DIR
cd $REPO_DIR

if [ -n "$SET_WD" ]
then
    SetWorkingDirectory "$REPO_DIR"
fi

if [ "$BRANCH" != "." ]
then
    CLONEREF="-b $BRANCH"
fi
git clone $CLONEREF $REPO_URL .
