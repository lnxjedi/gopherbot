#!/bin/bash

[ -z "$GOPHER_LOCALDIR" ] && { echo "GOPHER_LOCALDIR must be set"; exit 1; }

[ "$1" = "-f" ] && { GOPHER_MESSAGE_FORMAT="fixed"; shift; }

[ $# -lt 2 ] && { echo "Usage: sendusermsg.sh <channel> message"; exit 1; }
EXECPATH=$(dirname `readlink -f $0`)

source $EXECPATH/shellFuncs.sh

CHATUSER=$1
shift
MESSAGE="$*"

sendUserMessage $CHATUSER "$MESSAGE"
