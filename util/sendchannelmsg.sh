#!/bin/bash

[ $# -lt 2 ] && { echo "Usage: sendchannelmsg.sh <channel> message"; exit 1; }

CHANNEL=$1
shift
MESSAGE="$*"

JSON=$(cat <<EOF
{
	"Command": "SendChannelMessage",
	"CmdArgs": {
		"ChanID": "$CHANNEL",
		"Message": "$MESSAGE"
	}
}
EOF
)

echo "$JSON"
echo "$JSON" | curl -X POST -d @- -v http://localhost:8080/json
