#!/bin/bash

# cleanup.sh - task for removing the workdir at the end of a job.
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

CLEANUP="$1"

if [ -z "$CLEANUP" ]
then
    Log "Error" "Cleanup directory not given" >&2
    exit 1
else
    if [ -z "$GOPHER_WORKSPACE" ]
    then
        echo "GOPHER_WORKSPACE not set" >&2
        exit 1
    fi
    cd $GOPHER_WORKSPACE
    rm -rf "$CLEANUP"
fi
