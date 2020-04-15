#!/bin/bash -e

# startbuild.sh - utility task to tell the user a build is starting

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

TELL="Starting build of $GOPHER_REPOSITORY, branch: '$GOPHERCI_BRANCH'"

if [ "$GOPHER_LOG_REF" ] || [ "$GOPHER_LOG_LINK" ]
then
  if [ "$GOPHER_LOG_REF" ] && [ "$GOPHER_LOG_LINK" ]
  then
    TELL="$TELL (log $GOPHER_LOG_REF; link $GOPHER_LOG_LINK)"
  elif [ "$GOPHER_LOG_REF" ]
  then
    TELL="$TELL (log $GOPHER_LOG_REF)"
  else
    TELL="$TELL (link $GOPHER_LOG_LINK)"
  fi
fi

Say "$TELL"