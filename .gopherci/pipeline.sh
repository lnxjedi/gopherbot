#!/bin/bash

# pipeline.sh - trusted pipeline script for gopherci for Gopherbot

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -n "$NOTIFY_USER" ]
then
    FailTask notify $NOTIFY_USER "Gopherbot build failed"
fi

# Run tests
AddTask exec go test -v --tags 'test integration' -cover -race -coverprofile coverage.out -coverpkg ./... ./bot

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
    Say "Gosh, I wish that *I* could publish"
    FinalTask builtin-history history email "" $GOPHER_JOB_NAME:$GOPHER_NAMESPACE_EXTENDED $GOPHER_RUN_INDEX
    exit 0
fi

# Publish archives to github
AddTask exec ./.gopherci/publish.sh

# Trigger Docker build
AddTask exec curl -X POST https://cloud.docker.com/$DOCKER_TRIGGER

# Notify of success
if [ -n "$NOTIFY_USER" ]
then
    AddTask notify $NOTIFY_USER "Successfully built and released latest Gopherbot"
fi
