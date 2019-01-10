#!/bin/bash

# cleanup.sh - task for removing the workdir at the end of a job
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -z "$GOPHERCI_WORKDIR" ]
then
    echo "GOPHERCI_WORKDIR not set" >&2
    exit 1
fi

if [ -z "$1" ]
then
    SetWorkingDirectory "."
    AddTask $GOPHER_TASK_NAME $GOPHERCI_WORKDIR
    exit 0
fi

rm -rf "$1"
