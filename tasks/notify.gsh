#!/bin/sh

set -e

user=$1
message=$2

if [ -n "$GOPHER_CHANNEL" ]
then
	SendUserChannelMessage "$user" "$GOPHER_CHANNEL" "$message"
else
	SendUserMessage "$user" "$message"
fi
