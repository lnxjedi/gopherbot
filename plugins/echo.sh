#!/bin/bash -e

# echo.sh - trivial shell plugin example for Gopherbot
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/util/shellLib.sh

reply "$*"
