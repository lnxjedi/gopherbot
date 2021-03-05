#!/bin/bash

CTAG="latest"
if [ "$1" ]
then
    BUILDARG="--build-arg buildref=$1"
    CTAG="$1"
fi

docker build -f Containerfile $BUILDARG -t quay.io/lnxjedi/gopherbot:$CTAG .
