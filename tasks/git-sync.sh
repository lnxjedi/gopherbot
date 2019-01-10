#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ $# -lt 3 ]
then
    echo "Not enough arguments to git-sync; usage: git-sync <url> <branch> <dir> (true)" >&2
    exit 1
fi

# git-sync.sh - clone or update a git repository and optionally set the
# working directory
REPO_URL=$1
BRANCH=$2
REPO_DIR=$3
SET_WD=$4

mkdir -p $REPO_DIR
cd $REPO_DIR

if [ -e .git ]
then
    git fetch
    git checkout $BRANCH
    git pull
else
    git clone $REPO_URL .
    git checkout $BRANCH
fi

if [ -n "$SET_WD" ]
then
    SetWorkingDirectory "$REPO_DIR"
    SetParameter "GOPHERCI_WORKDIR" "$REPO_DIR"
fi
