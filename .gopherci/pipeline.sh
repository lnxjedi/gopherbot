#!/bin/bash

# pipeline.sh - trusted pipeline script for gopherci for Gopherbot

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

# Get dependencies
AddTask localexec go get -v -t -d ./...

# Run tests
AddTask localexec go test -v --tags 'test integration' -cover -race -coverprofile coverage.out -coverpkg ./... ./bot

# Install required tools
AddTask localexec ./.gopherci/tools.sh

# Publish coverage results
AddTask goveralls -coverprofile=coverage.out -service=circle-ci -repotoken=$COVERALLS_TOKEN

# Do a full build for all platforms
AddTask localexec ./mkdist.sh

# Publish archives to github
AddTask localexec ./.gopherci/publish.sh