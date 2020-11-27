#!/bin/bash

# pipeline.sh - trusted pipeline script for gopherci for Gopherbot

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -n "$NOTIFY_USER" ]
then
    FailTask notify $NOTIFY_USER "Gopherbot build failed"
fi

FailTask email-log parsley@linuxjedi.org

# Run tests
AddTask exec go test -v --tags 'test integration netgo osusergo static_build' -mod vendor -cover -race -coverprofile coverage.out -coverpkg ./... ./test

# Do a full build
AddTask exec make

# See who got this message and act accordingly
BOT=$(GetBotAttribute name)
if [ "$BOT" != "data" ]
then
    # if it's not Data, stop the pipeline here
    if [ -n "$NOTIFY_USER" ]
    then
        AddTask notify $NOTIFY_USER "Builds and tests succeeded for Gopherbot"
    else
        Say "NOTIFY_USER not set"
    fi
    exit 0
fi

if [ "$GOPHERCI_BRANCH" != "master" -o "$GOPHER_REPOSITORY" == "github.com/parsley42/gopherbot" ]
then
    AddTask notify $NOTIFY_USER "Completed successful build and test of $GOPHER_REPOSITORY branch $GOPHERCI_BRANCH"
    exit 0
fi

# Set for building containers in containers
SetParameter BUILDAH_ISOLATION chroot

# Log in to container registries
AddTask buildah-login quay.io parsley42 QUAY
AddTask buildah-login registry.in.linuxjedi.org linux LINUXJEDI

# Build the containers, tag for developer registry
# Note that the make target pulls the FROM images first
AddTask exec make containers
AddTask exec buildah tag quay.io/lnxjedi/gopherbot registry.in.linuxjedi.org/lnxjedi/gopherbot
AddTask exec buildah tag quay.io/lnxjedi/gopherbot-theia registry.in.linuxjedi.org/lnxjedi/gopherbot-theia

# Push containers out
AddTask exec buildah push quay.io/lnxjedi/gopherbot
AddTask exec buildah push quay.io/lnxjedi/gopherbot-theia
AddTask exec buildah push registry.in.linuxjedi.org/lnxjedi/gopherbot
AddTask exec buildah push registry.in.linuxjedi.org/lnxjedi/gopherbot-theia

# Notify of success
if [ -n "$NOTIFY_USER" ]
then
    AddTask notify $NOTIFY_USER "Successfully built and pushed gopherbot:latest"
fi
