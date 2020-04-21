#!/bin/bash -e

# runpipeline.sh - utility task to detect and run the repository
# pipeline, or custom job pipeline

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

PIPELINE=${1}
shift

for PTRY in $PIPELINE $PIPELINE.sh $PIPELINE.py $PIPELINE.rb
do
    if [ -x ".gopherci/$PTRY" ]
    then
        AddTask exec "./.gopherci/$PTRY $*"
        exit 0
    fi
done

Log "Warn" "Repository pipeline not found in job $GOPHER_JOB_NAME (wd: $(pwd), repo: ${GOPHER_REPOSITORY:-not set}), ignoring"
