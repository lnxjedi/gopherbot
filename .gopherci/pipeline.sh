#!/bin/bash

# pipeline.sh - trusted pipeline script for gopherci for Gopherbot

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -n "$NOTIFY_USER" ]
then
    FailTask notify $NOTIFY_USER "Gopherbot build failed"
fi

# Update path for a Go build
PATH=$PATH:$HOME/go/bin:/usr/local/go/bin
SetParameter "PATH" "$PATH"

# Email the job history if it fails
FailCommand builtin-history "send history $GOPHER_JOB_NAME:$GOPHER_NAMESPACE_EXTENDED/$GOPHERCI_BRANCH $GOPHER_RUN_INDEX to user parsley"

# Run tests
AddTask exec go test -v --tags 'test integration netgo osusergo static_build' -mod vendor -cover -race -coverprofile coverage.out -coverpkg ./... ./bot

# Install required tools
AddTask exec ./.gopherci/tools.sh

# Publish coverage results
#AddTask exec goveralls -coverprofile=coverage.out -service=circle-ci -repotoken=$COVERALLS_TOKEN

# Do a full build for all platforms
AddTask exec ./.gopherci/mkdist.sh

# See who got this message and decide whether to build
BOT=$(GetBotAttribute name)
if [ "$BOT" != "floyd" ]
then
    if [ -n "$NOTIFY_USER" ]
    then
        AddTask notify $NOTIFY_USER "Builds and tests succeeded for Gopherbot"
    else
        Say "NOTIFY_USER not set"
    fi
    exit 0
fi

Say "GOPHERCI_REPO is $GOPHERCI_REPO"
Log "Warn" "GOPHERCI_REPO is $GOPHERCI_REPO"

exit 0

if [ "$GOPHERCI_BRANCH" != "master" -o "$GOPHERCI_REPO" == "github.com/parsley42/gopherbot" ]
then
    AddTask notify $NOTIFY_USER "Completed successful build and test of $GOPHERCI_REPO branch $GOPHERCI_BRANCH"
    exit 0
fi

# Publish archives to github
#AddTask exec ./.gopherci/publish.sh

# Trigger Docker build
#AddTask exec ./.gopherci/dockercloud.sh

# Notify of success
if [ -n "$NOTIFY_USER" ]
then
    AddTask notify $NOTIFY_USER "Successfully built and released latest Gopherbot"
fi
