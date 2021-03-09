#!/bin/bash

# get-version.sh - used by the Makefile to inject the correct
# version in to the Go linker.

if [ "$BUILDREF" ]
then
    echo "$BUILDREF"
    exit 0
fi

VERLINE=$(grep "^var Version" main.go)
VERLINE=${VERLINE%\"*}
VERSION=${VERLINE##*\"}
echo "$VERSION"
