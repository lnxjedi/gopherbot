#!/bin/bash -e

# test/tasks/pipeline_note.sh - helper for pipeline control integration tests.

source "$GOPHER_INSTALLDIR/lib/gopherbot_v1.sh"

TAG="$*"
if [ "$TAG" = "spawn-step" ]; then
  # Keep spawn-job output ordered after the parent command's queue message.
  sleep 1
fi

Say "PIPE NOTE: $TAG"
