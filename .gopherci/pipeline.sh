#!/bin/bash

# pipeline.sh - trusted pipeline script for gopherci for Gopherbot

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

PATH=$HOME/go/bin:$PATH:/usr/local/go/bin
# Add go binaries to PATH for the rest of the pipeline
SetParameter PATH "$PATH"

if [-n "$NOTIFY_USER"]
then
    FailTask notify $NOTIFY_USER "Gopherbot build failed"
fi

# Get dependencies
AddTask localexec go get -v -t -d ./...

# Run tests
AddTask localexec go test -v --tags 'test integration' -cover -race -coverprofile coverage.out -coverpkg ./... ./bot

# Install required tools
AddTask localexec ./.gopherci/tools.sh

# Publish coverage results
#AddTask localexec goveralls -coverprofile=coverage.out -service=circle-ci -repotoken=$COVERALLS_TOKEN

# Do a full build for all platforms
AddTask localexec ./mkdist.sh

# Publish archives to github
AddTask localexec ./.gopherci/publish.sh

# Notify of success
if [ -n "$NOTIFY_USER" ]
then
    AddTask notify $NOTIFY_USER "Successfully built and released latest Gopherbot"
fi
