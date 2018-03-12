#!/bin/bash

# publish.sh - copy the install archive to a distribution point

VERSTRING=$(grep "var Version" bot/bot.go)
VERSTRING=${VERSTRING#var }
VERSTRING=${VERSTRING// /}
eval $VERSTRING
eval `go env`
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ $BRANCH = "master" ]
then
    TAG=$Version
    RELEASE="$Version-stable"
else
    TAG="$BRANCH-snapshot"
    RELEASE=$TAG
    PRERELEASE="--prerelease"
fi

github-release $RELEASE *zip --github-repository lnxjedi/gopherbot --tag $TAG --replace $PRERELEASE