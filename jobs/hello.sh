#!/bin/bash -e

# jobs/hello.sh - the first Gopherbot scheduled job

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

case "$COMMAND" in
  "hello")
    SendChannelMessage "general" "Hello, World!"
    # this should be saved in job history
    echo "this job succeeded"
    ;;
  "goodbye")
    Pause 7
    SendChannelMessage "general" "So long!"
    ;;
  "fail")
    echo "about to fail..."
    # this also in job history, prefixed with STDERR
    echo "this job fails..." >&2
    exit 1
    ;;
esac