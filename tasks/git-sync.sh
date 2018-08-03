#!/bin/bash

# git-sync.sh - clone or update a git repository
REPO_URL=$1
REPO_DIR=$2

mkdir -p $REPO_DIR
cd $REPO_DIR

if [ -e .git ]
then
    git pull
else
    git clone $REPO_URL .
fi