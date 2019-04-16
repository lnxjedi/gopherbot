#!/bin/bash

# cleanup.sh - task for removing the workdir at the end of a job.
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -z "$GOPHER_JOB_DIR" ]
then
    echo "GOPHER_JOB_DIR not set" >&2
    exit 1
else
    if [ -z "$GOPHER_WORKSPACE" ]
    then
        echo "GOPHER_WORKSPACE not set" >&2
        exit 1
    fi
    cd $GOPHER_WORKSPACE
    rm -rf "$GOPHER_JOB_DIR"
fi
