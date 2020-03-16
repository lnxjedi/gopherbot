#!/bin/bash -e

# tasks/status.sh - trivial task that can be used to send status updates
# in a pipeline.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

MESSAGE="$1"
Say "$MESSAGE"
