#!/bin/bash -e

# test/tasks/param_show.sh - integration helper for verifying that
# pipeline SetParameter values are visible to subsequent AddTask runs.

source "$GOPHER_INSTALLDIR/lib/gopherbot_v1.sh"

VALUE="${PIPELINE_SENTINEL:-}"
if [ -z "$VALUE" ]; then
  VALUE="<empty>"
fi

Say "PARAM-SHOW: PIPELINE_SENTINEL=$VALUE"
