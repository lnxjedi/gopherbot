#!/bin/bash

# publish.sh - copy the install archive to a distribution point

VERSTRING=$(grep "var Version" bot/bot.go)
VERSTRING=${VERSTRING#var }
VERSTRING=${VERSTRING// /}
eval $VERSTRING
eval `go env`
BRANCH=$(git rev-parse --abbrev-ref HEAD)
COMMIT=$(git rev-parse HEAD)
if [ $BRANCH = "master" ]
then
    RELEASE=$Version
    if [[ $RELEASE = *-snapshot ]]
    then
        PRERELEASE="--prerelease"
        UPDATE="--update"
    fi
else
    RELEASE="$BRANCH-snapshot"
    PRERELEASE="--prerelease"
    UPDATE="--update"
fi

github-release $RELEASE *zip --github-repository lnxjedi/gopherbot --commit $COMMIT --target $BRANCH $PRERELEASE $UPDATE
