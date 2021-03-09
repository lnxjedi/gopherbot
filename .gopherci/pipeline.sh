#!/bin/bash

# pipeline.sh - trusted pipeline script for gopherci for Gopherbot

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -n "$NOTIFY_USER" ]
then
    FailTask notify $NOTIFY_USER "Gopherbot build failed"
fi

FailTask email-log parsley@linuxjedi.org

CTAG="latest"

if [[ $GOPHERCI_BRANCH == release-* ]]
then
    if [ "$GOPHER_PIPELINE_TYPE" != "plugCommand" ]
    then
        Say "Skipping build of $GOPHER_REPOSITORY, ref '$GOPHERCI_BRANCH' (requires manual build)"
        exit 0
    fi
    CTAG="$GOPHERCI_BRANCH"
    SetParameter BUILDREF "$GOPHERCI_BRANCH"
fi

if [[ $GOPHERCI_BRANCH == v*.* ]]
then
    if [ "$GOPHER_PIPELINE_TYPE" != "plugCommand" ]
    then
        Say "Skipping build of $GOPHER_REPOSITORY, ref '$GOPHERCI_BRANCH' (requires manual build)"
        exit 0
    fi
    CTAG="$GOPHERCI_BRANCH"
    SetParameter BUILDREF "$GOPHERCI_BRANCH"
fi

# SetParameter ~= "export" for the pipeline.
SetParameter CTAG "$CTAG"

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

# Set for building containers in containers
SetParameter BUILDAH_ISOLATION chroot

# Log in to container registries
AddTask buildah-login quay.io parsley42 QUAY
AddTask buildah-login registry.in.linuxjedi.org linux LINUXJEDI

# Build the containers, tag for developer registry
# Note that the make target pulls the FROM images first
AddTask exec make containers
AddTask exec buildah tag quay.io/lnxjedi/gopherbot:$CTAG registry.in.linuxjedi.org/lnxjedi/gopherbot
AddTask exec buildah tag quay.io/lnxjedi/gopherbot-theia:$CTAG registry.in.linuxjedi.org/lnxjedi/gopherbot-theia
AddTask exec buildah tag quay.io/lnxjedi/gopherbot-dev:$CTAG registry.in.linuxjedi.org/lnxjedi/gopherbot-dev

# Push containers out
AddTask exec buildah push quay.io/lnxjedi/gopherbot:$CTAG
AddTask exec buildah push quay.io/lnxjedi/gopherbot-theia:$CTAG
AddTask exec buildah push quay.io/lnxjedi/gopherbot-dev:$CTAG
AddTask exec buildah push registry.in.linuxjedi.org/lnxjedi/gopherbot
AddTask exec buildah push registry.in.linuxjedi.org/lnxjedi/gopherbot-theia
AddTask exec buildah push registry.in.linuxjedi.org/lnxjedi/gopherbot-dev
# As good a place as any for now? Need to remove later in favor of weekly job.
AddTask exec buildah rmi -p

# Notify of success
if [ -n "$NOTIFY_USER" ]
then
    AddTask notify $NOTIFY_USER "Successfully built and pushed gopherbot:latest"
fi
