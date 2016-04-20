#!/bin/bash

[ -z "$GOPHER_LOCALDIR" ] && { echo "GOPHER_LOCALDIR must be set"; exit 1; }

[ "$1" = "-f" ] && { GOPHER_MESSAGE_FORMAT="fixed"; shift; }

[ $# -lt 3 ] && { echo "Usage: senduserchannelmsg.sh <user> <channel> message"; exit 1; }
EXECPATH=$(dirname `readlink -f $0`)

source $EXECPATH/shellFuncs.sh

CHATUSER=$1
CHANNEL=$2
shift 2
MESSAGE="$*"

sendUserChannelMessage $CHATUSER $CHANNEL "$MESSAGE"
