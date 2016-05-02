gb_json_encode(){
	local MESSAGE
	MESSAGE=$(echo "$@" | base64)
	MESSAGE=$(echo "base64:$MESSAGE")
	echo "$MESSAGE"
}

GetAttribute(){
	local JSON
	local ATTR="$1"
	JSON=$(cat <<EOF
{
	"Command": "GetAttribute",
	"PluginID": "$GB_PLUGID",
	"CmdArgs": {
		"Attribute": "$ATTR"
	}
}
EOF
)
	echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null
}

# Post the JSON and get error output
gbPostJSON(){
	local JSON="$1"
	local OUTPUTVAR=$2
	local OUTTEMP=$(mktemp /tmp/gopherpost-XXXXXX)
	local HEADTEMP=$(mktemp /tmp/gopherhead-XXXXXX)
	local OUTPUT
	echo "$JSON" | curl -f -X POST -D $HEADTEMP -d @- $GOPHER_HTTP_POST/json 2>/dev/null > $OUTTEMP
	echo "Headers in $HEADTEMP" >&2
	OUTPUT=$(cat $OUTTEMP)
	rm -f $OUTTEMP
	eval $OUTPUTVAR=\"$OUTPUT\"
}

GetUserAttribute(){
	local JSON
	local ATTR="$1"
	JSON=$(cat <<EOF
{
	"Command": "GetUserAttribute",
	"PluginID": "$GB_PLUGID",
	"CmdArgs": {
		"User": "$GB_USER",
		"Attribute": "$ATTR"
	}
}
EOF
)
	echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null
}

WaitForReply(){
	local JSON
	local REGEX="$1"
	local TIMEOUT="${2:-14}"
	local NEEDCMD="${3:-false}"
	JSON=$(cat <<EOF
{
	"Command": "WaitForReply",
	"PluginID": "$GB_PLUGID",
	"CmdArgs": {
		"User": "$GB_USER",
		"Channel": "$GB_CHANNEL",
		"RegExId": "$REGEX",
		"Timeout": $TIMEOUT,
		"NeedCommand": $NEEDCMD
	}
}
EOF
)
	echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null
}

SendUserMessage(){
	local JSON
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	GB_FORMAT=${GB_FORMAT:-variable}
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	JSON=$(cat <<EOF
{
	"Command": "SendUserMessage",
	"PluginID": "$GB_PLUGID",
	"CmdArgs": {
		"User": "$GB_USER",
		"Format": "$GB_FORMAT",
		"Message": "$MESSAGE"
	}
}
EOF
)
	echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null
}

SendUserChannelMessage(){
	local JSON
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	GB_FORMAT=${GB_FORMAT:-variable}
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	JSON=$(cat <<EOF
{
	"Command": "SendUserChannelMessage",
	"PluginID": "$GB_PLUGID",
	"CmdArgs": {
		"User": "$GB_USER",
		"Channel": "$GB_CHANNEL",
		"Format": "$GB_FORMAT",
		"Message": "$MESSAGE"
	}
}
EOF
)
	echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null
}

SendChannelMessage(){
	local JSON
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	GB_FORMAT=${GB_FORMAT:-variable}
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

JSON=$(cat <<EOF
{
	"Command": "SendChannelMessage",
	"PluginID": "$GB_PLUGID",
	"CmdArgs": {
		"Channel": "$GB_CHANNEL",
		"Format": "$GB_FORMAT",
		"Message": "$MESSAGE"
	}
}
EOF
)
	echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null
}

# Convenience functions so that copies of this logic don't wind up in a bunch of plugins
Say(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendChannelMessage "$*"
	else
		SendUserMessage "$*"
	fi
}

Reply(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendUserChannelMessage "$*"
	else
		SendUserMessage "$*"
	fi
}
