#!/bin/bash

# cleanup.sh - task for removing the workdir at the end of a job

rm -rf "$GOPHERCI_WORKDIR"
