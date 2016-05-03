gb_json_encode(){
	local MESSAGE
	MESSAGE=$(echo "$@" | base64)
	MESSAGE=$(echo "base64:$MESSAGE")
	echo "$MESSAGE"
}

# Create the full JSON string and post it
gbPostJSON(){
	local GB_FUNCNAME=$1
	local GB_FUNCARGS="$2"
	local JSON
	GB_FORMAT=${GB_FORMAT:-variable}
	JSON=$(cat <<EOF
{
	"FuncName": "$GB_FUNCNAME",
	"User": "$GB_USER",
	"Channel": "$GB_CHANNEL",
	"Format": "$GB_FORMAT",
	"PluginID": "$GB_PLUGID",
	"FuncArgs": $GB_FUNCARGS
}
EOF
)
	echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null
}

GetAttribute(){
	local GB_FUNCARGS
	local GB_FUNCNAME="GetAttribute"
	local ATTR="$1"
	GB_FUNCARGS=$(cat <<EOF
{
	"Attribute": "$ATTR"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
}

GetUserAttribute(){
	local GB_FUNCARGS
	local GB_FUNCNAME="GetUserAttribute"
	local ATTR="$1"
	GB_FUNCARGS=$(cat <<EOF
{
	"Attribute": "$ATTR"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
}

WaitForReply(){
	local GB_FUNCARGS
	local GB_FUNCNAME="WaitForReply"
	local REGEX="$1"
	local TIMEOUT="${2:-14}"
	local NEEDCMD="${3:-false}"
	GB_FUNCARGS=$(cat <<EOF
{
	"RegExId": "$REGEX",
	"Timeout": $TIMEOUT,
	"NeedCommand": $NEEDCMD
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
}

SendMessage(){
	local GB_FUNCARGS
	local GB_FUNCNAME=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"Message": "$MESSAGE"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
}

SendUserChannelMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	SendMessage "SendUserChannelMessage" "$*"
}

SendUserMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	SendMessage "SendUserMessage" "$*"
}

SendChannelMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	SendMessage "SendChannelMessage" "$*"
}

# Convenience functions so that copies of this logic don't wind up in a bunch of plugins
Say(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendMessage "SendChannelMessage" "$*"
	else
		SendMessage "SendUserMessage" "$*"
	fi
}

Reply(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendMessage "SendUserChannelMessage" "$*"
	else
		SendMessage "SendUserMessage" "$*"
	fi
}
