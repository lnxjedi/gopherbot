#!/bin/bash

# cleanup.sh - task for removing the workdir at the end of a job;
# pipeline needs to reset workdir first
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -z "$GOPHERCI_WORKDIR" ]
then
    echo "GOPHERCI_WORKDIR not set" >&2
    exit 1
else
    rm -rf "$GOPHERCI_WORKDIR"
fi
