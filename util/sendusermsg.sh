#!/bin/bash

[ $# -lt 2 ] && { echo "Usage: sendusermsg.sh <user> message"; exit 1; }

CHATUSER=$1
shift
MESSAGE="$*"

JSON=$(cat <<EOF
{
	"Command": "SendUserMessage",
	"CmdArgs": {
		"User": "$CHATUSER",
		"Message": "$MESSAGE"
	}
}
EOF
)

echo "$JSON"
echo "$JSON" | curl -X POST -d @- -v http://localhost:8880/json
