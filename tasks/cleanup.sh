#!/bin/bash

# cleanup.sh - task for removing the workdir at the end of a job

cd "$GOPHER_WORKSPACE"
rm -rf "$GOPHERCI_WORKDIR"
