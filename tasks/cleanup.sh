#!/bin/bash

# cleanup.sh - task for cleaning a workdir at the start of a job.
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh
if [ -z "$GOPHER_WORKDIR" ]
then
    echo "GOPHER_WORKDIR not set" >&2
    exit 1
fi
if [[ $GOPHER_WORKDIR = /* ]]
then
    Log "Error" "Not cleaning absolute GOPHER_WORKDIR: $GOPHER_WORKDIR"
    exit 1
fi
rm -rf "$GOPHER_WORKDIR"
mkdir -p "$GOPHER_WORKDIR"
