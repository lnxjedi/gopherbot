#!/bin/bash

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

OLD="log.$(date +%a)"
AddTask "rotate-log" "$OLD"
