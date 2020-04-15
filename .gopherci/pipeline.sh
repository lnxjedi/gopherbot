#!/bin/bash

# pipeline.sh - trusted pipeline script for gopherci for Gopherbot

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -n "$NOTIFY_USER" ]
then
    FailTask notify $NOTIFY_USER "Gopherbot build failed"
fi

REPO_NAME=${GOPHER_NAMESPACE_EXTENDED:-$GOPHER_REPOSITORY}

# Update path for a Go build
PATH=$PATH:$HOME/go/bin:/usr/local/go/bin
SetParameter "PATH" "$PATH"

# Do a full build for all platforms
AddTask exec ./.gopherci/mkdist.sh

# Run tests
AddTask exec go test -v --tags 'test integration netgo osusergo static_build' -mod vendor -cover -race -coverprofile coverage.out -coverpkg ./... ./test

# Install required tools
AddTask exec ./.gopherci/tools.sh

# Publish coverage results
#AddTask exec goveralls -coverprofile=coverage.out -service=circle-ci -repotoken=$COVERALLS_TOKEN

# Initial clones from public https
AddTask git-clone https://github.com/lnxjedi/gopherbot.git gh-pages gopherbot-doc
AddTask git-clone https://github.com/lnxjedi/gopherbot-docker.git master gopherbot-docker

AddTask exec ./.gopherci/mkdocs.sh

# See who got this message and act accordingly
BOT=$(GetBotAttribute name)
if [ "$BOT" != "floyd" ]
then
    if [ -n "$NOTIFY_USER" ]
    then
        AddTask notify $NOTIFY_USER "Builds and tests succeeded for Gopherbot"
# Email the job history if it fails
FailCommand builtin-history "send history $GOPHER_JOB_NAME:$REPO_NAME/$GOPHERCI_BRANCH $GOPHER_RUN_INDEX to user parsley"
    else
        Say "NOTIFY_USER not set"
    fi
    FailTask email-log parsley@linuxjedi.org
    exit 0
else
    # Email the job history if it fails
    FailCommand builtin-history "send history $GOPHER_JOB_NAME:$REPO_NAME/$GOPHERCI_BRANCH $GOPHER_RUN_INDEX to user parsley"
fi

if [ "$GOPHERCI_BRANCH" != "master" -o "$GOPHER_REPOSITORY" == "github.com/parsley42/gopherbot" ]
then
    AddTask notify $NOTIFY_USER "Completed successful build and test of $GOPHER_REPOSITORY branch $GOPHERCI_BRANCH"
    exit 0
fi

# Initialize ssh for updating docs repo
AddTask ssh-init

# Make sure github is in known_hosts
AddTask ssh-scan github.com

# Publish doc updates (if any)
AddTask exec ./.gopherci/publishdoc.sh

# Publish archives to github
AddTask exec ./.gopherci/publish.sh

# Update gopherbot-docker
AddTask exec ./.gopherci/dockerupdate.sh

# Notify of success
if [ -n "$NOTIFY_USER" ]
then
    AddTask notify $NOTIFY_USER "Successfully built and released latest Gopherbot"
fi
