#!/bin/bash -e

# tasks/reply.sh - trivial task that can be used to reply to the user
# in a pipeline.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ "$GOPHER_USER" ]
then
    Reply "$*"
else
    Say "$*"
fi
