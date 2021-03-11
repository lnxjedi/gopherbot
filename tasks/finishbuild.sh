#!/bin/bash -e

# finishbuild.sh - utility task to tell the user a build has finished

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ "$GOPHERCI_BRANCH" == "." ]
then
    TELL_BRANCH="(default branch)"
else
    TELL_BRANCH="$GOPHERCI_BRANCH"
fi

if [ "$GOPHER_FAIL_CODE" ]
then
  if [ "$GOPHER_FINAL_TYPE" == "plugin" ]
  then
    FAILED="plugin $GOPHER_FINAL_TASK, command \"$GOPHER_FINAL_COMMAND\""
  else
    FAILED="$GOPHER_FINAL_TYPE $GOPHER_FINAL_TASK"
  fi
  if [ "$GOPHER_FINAL_ARGS" ]
  then
    FAILED="$FAILED with args: $GOPHER_FINAL_ARGS"
  fi
  REF=";"
  if [ "$GOPHER_LOG_REF" ]
  then
    if [ "$GOPHER_LOG_LINK" ]
    then
      REF=" (log $GOPHER_LOG_REF: $GOPHER_LOG_LINK);"
    else
      REF=" (log $GOPHER_LOG_REF);"
    fi
  fi
  if [ "$GOPHERCI_CUSTOM_PIPELINE" ]
  then
    TELL="JOB FAILED for $GOPHER_REPOSITORY, branch '$TELL_BRANCH'$REF, running pipeline '$GOPHERCI_CUSTOM_PIPELINE': failure in $FAILED; exit code $GOPHER_FAIL_CODE ($GOPHER_FAIL_STRING)"
  else
    TELL="BUILD FAILED for $GOPHER_REPOSITORY, branch '$TELL_BRANCH'$REF: failure in $FAILED; exit code $GOPHER_FAIL_CODE ($GOPHER_FAIL_STRING)"
  fi
else
  if [ "$GOPHERCI_CUSTOM_PIPELINE" ]
  then
    TELL="Custom job for $GOPHER_REPOSITORY, branch '$TELL_BRANCH', running pipeline '$GOPHERCI_CUSTOM_PIPELINE' finished successfully"
  else
    TELL="Build of $GOPHER_REPOSITORY, branch '$TELL_BRANCH' finished successfully"
  fi
fi

Say "$TELL"