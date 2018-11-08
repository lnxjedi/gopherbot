#!/bin/bash -e

# tasks/fail.sh - used by gopherci to indicate a pipeline failing while
# allowing previous tasks to complete

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

MESSAGE=$1
Log "Error" "$MESSAGE"
exit 1