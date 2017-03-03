#!/bin/bash
# shellLib.sh - bash plugins should source this with 'source $GOPHER_INSTALLDIR/util/shellLib.sh'
GBRET_Ok=0
GBRET_UserNotFound=1
GBRET_ChannelNotFound=2
GBRET_AttributeNotFound=3
GBRET_FailedUserDM=4
GBRET_FailedChannelJoin=5
GBRET_DatumNotFound=6
GBRET_DatumLockExpired=7
GBRET_DataFormatError=8
GBRET_BrainFailed=9
GBRET_InvalidDatumKey=10
GBRET_InvalidDblPtr=11
GBRET_InvalidCfgStruct=12
GBRET_NoConfigFound=13
GBRET_NoUserOTP=14
GBRET_OTPError=15
GBRET_ReplyNotMatched=16
GBRET_UseDefaultValue=17
GBRET_TimeoutExpired=18
GBRET_ReplyInProgress=19
GBRET_MatcherNotFound=20
GBRET_NoUserEmail=21
GBRET_NoBotEmail=22
GBRET_MailError=23

GB_CHANNEL=$1
GB_USER=$2
GB_PLUGID=$3
export GB_CHANNEL GB_USER GB_PLUGID
shift 3
# Now $1 is the command

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
	local JSON JSONRET
	#local GB_DEBUG="true"
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
	if [ "$GB_DEBUG" = "true" ]
	then
		echo "Sending:" >&2
		echo "$JSON" >&2
	fi
	JSONRET=$(echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null)
	if [ "$GB_DEBUG" = "true" ]
	then
		echo "Got back:" >&2
		echo "$JSONRET" >&2
	fi
	echo "$JSONRET"
	GB_FORMAT="variable"
}

gbBotRet() {
	local JSON="$1"
	local RETVAL
	RETVAL=$(echo "$JSON" | jq .RetVal)
	return $RETVAL
}

gbDecode() {
	local JSON="$1"
	local ITEM="$2"
	local B64DATA=$(echo "$JSON" | jq -r .$ITEM)
	B64DATA=${B64DATA#base64:}
	echo "$B64DATA" | base64 -d
}

CheckAdmin(){
	local GB_FUNCARGS="{}"
	local GB_FUNCNAME="CheckAdmin"
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	local RETVAL=$(echo "$GB_RET" | jq .Boolean)
	echo "$RETVAL"
	if [ "$RETVAL" -eq "true" ]
	then
		return 0
	else
		return 1
	fi
}

CheckOTP() {
	local GB_FUNCARGS GB_RET BOOL RETVAL
	local GB_FUNCNAME="CheckOTP"
	local CODE="$1"
	GB_FUNCARGS=$(cat <<EOF
{
	"Code": "$CODE"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	BOOL=$(echo "$GB_RET" | jq .Boolean)
	RETVAL=$(echo "$GB_RET" | jq .RetVal)
	echo "$BOOL"
	return $RETVAL
}

GetBotAttribute(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="GetBotAttribute"
	local ATTR="$1"
	GB_FUNCARGS=$(cat <<EOF
{
	"Attribute": "$ATTR"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Attribute
	gbBotRet "$GB_RET"
}

GetSenderAttribute(){
	local GB_FUNCARGS
	local GB_FUNCNAME="GetSenderAttribute"
	local ATTR="$1"
	GB_FUNCARGS=$(cat <<EOF
{
	"Attribute": "$ATTR"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Attribute
	gbBotRet "$GB_RET"
}

GetUserAttribute(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="GetUserAttribute"
	local GUA_USER="$1"
	local ATTR="$2"
	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$GUA_USER",
	"Attribute": "$ATTR"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Attribute
	gbBotRet "$GB_RET"
}

Log(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="Log"
	local GLM_LEVEL="$1"
	local GLM_MESSAGE="$2"
	GB_FUNCARGS=$(cat <<EOF
{
	"Level": "$GLM_LEVEL",
	"Message": "$GLM_MESSAGE"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

WaitForReply(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="WaitForReply"
	local REGEX="$1"
	local TIMEOUT="${2:-14}"
	GB_FUNCARGS=$(cat <<EOF
{
	"RegExId": "$REGEX",
	"Timeout": $TIMEOUT
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Reply
	gbBotRet "$GB_RET"
}

WaitForReplyRegex(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="WaitForReplyRegex"
	local REGEX="$1"
	local TIMEOUT="${2:-14}"
	GB_FUNCARGS=$(cat <<EOF
{
	"RegEx": "$REGEX",
	"Timeout": $TIMEOUT
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Reply
	gbBotRet "$GB_RET"
}

SendUserMessage(){
	if [ "$1" = "-f" ]; then GB_FORMAT=fixed; shift; else GB_FORMAT=variable; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendUserMessage"
	local SUM_USER=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$SUM_USER",
	"Message": "$MESSAGE"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

SendUserChannelMessage(){
	if [ "$1" = "-f" ]; then GB_FORMAT=fixed; shift; else GB_FORMAT=variable; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendUserChannelMessage"
	local SUCM_USER=$1
	local SUCM_CHANNEL=$2
	shift 2
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$SUCM_USER",
	"Channel": "$SUCM_CHANNEL",
	"Message": "$MESSAGE"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

SendChannelMessage(){
	if [ "$1" = "-f" ]; then GB_FORMAT=fixed; shift; else GB_FORMAT=variable; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendChannelMessage"
	local SCM_CHANNEL=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"Channel": "$SCM_CHANNEL",
	"Message": "$MESSAGE"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

# Convenience functions so that copies of this logic don't wind up in a bunch of plugins
Say(){
	local FARG
	[ "$1" = "-f" ] && { FARG="-f"; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendChannelMessage $FARG "$GB_CHANNEL" "$*"
	else
		SendUserMessage $FARG "$GB_USER" "$*"
	fi
}

Reply(){
	local FARG
	[ "$1" = "-f" ] && { FARG="-f"; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendUserChannelMessage $FARG "$GB_USER" "$GB_CHANNEL" "$*"
	else
		SendUserMessage $FARG "$GB_USER" "$*"
	fi
}
