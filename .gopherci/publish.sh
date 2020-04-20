#!/bin/bash

# publish.sh - copy the install archive to a distribution point

VERSION=$(grep "Version =" main.go)
VERSION=${VERSION#*= \"}
VERSION=${VERSION%\"}

eval `go env`
BRANCH=$(git rev-parse --abbrev-ref HEAD)
COMMIT=$(git rev-parse HEAD)
if [ $BRANCH = "master" ] || [[ $BRANCH = *-release ]]
then
    RELEASE=$VERSION
    if [[ $RELEASE = *-snapshot ]]
    then
        PRERELEASE="--prerelease"
        UPDATE="--update"
    elif [[ $RELEASE = *-beta* ]]
    then
        PRERELEASE="--prerelease"
    fi
fi

github-release $RELEASE *zip *tar.gz --github-repository lnxjedi/gopherbot --commit $COMMIT --target $BRANCH $PRERELEASE $UPDATE
