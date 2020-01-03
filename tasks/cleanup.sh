#!/bin/bash

# cleanup.sh - task for cleaning a workdir at the start of a job.
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

WORKDIR="$1"

if [ ! "$WORKDIR" ]
then
    Log "Error" "Argument WORKDIR not given" >&2
    exit 1
fi
if [[ $WORKDIR = /* ]]
then
    Log "Error" "Not cleaning absolute WORKDIR: $WORKDIR"
    exit 1
fi
if [ ! -d "$WORKDIR" ]
then
    Log "Info" "WORKDIR: $WORKDIR not found, ignoring"
    exit 0
fi
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
