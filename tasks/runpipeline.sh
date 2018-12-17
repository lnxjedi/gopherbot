#!/bin/bash -e

# runpipeline.sh - utility task to detect and run the repository
# pipeline

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

for PTRY in pipeline.sh pipeline.py pipeline.rb
do
    if [ -x ".gopherci/$PTRY" ]
    then
        AddTask exec "./.gopherci/$PTRY"
        exit 0
    fi
done

Log "Warn" "Repository pipeline not found in job $GOPHER_JOB_NAME (wd: $(pwd), repo: ${GOPHERCI_REPO:-not set}), ignoring"
