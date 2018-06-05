#!/bin/bash -e

# jobs/goodbye.sh - the first Gopherbot pipeline job

# NOTE: this sample job uses the bot library, most jobs probably won't
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

Pause 7
# Required parameter
Say "$PHRASE"