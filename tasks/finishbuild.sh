#!/bin/bash -e

# finishbuild.sh - utility task to tell the user a build has finished

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ "$GOPHER_FAIL_CODE" ]
then
  if [ "$GOPHER_FINAL_TYPE" == "plugin" ]
  then
    FAILED="plugin $GOPHER_FINAL_TASK, command \"$GOPHER_FINAL_COMMAND\""
  else
    FAILED="$GOPHER_FINAL_TYPE"
  fi
  if [ "$GOPHER_FINAL_ARGS" ]
  then
    FAILED="$FAILED with args: $GOPHER_FINAL_ARGS"
  fi
  TELL="Build failed for $GOPHER_REPOSITORY, branch: '$GOPHERCI_BRANCH'; failure in $FAILED; exit code $GOPHER_FAIL_CODE ($GOPHER_FAIL_STRING)"
else
  TELL="Build of $GOPHER_REPOSITORY, branch: '$GOPHERCI_BRANCH' finished successfully"
fi

Say "$TELL"