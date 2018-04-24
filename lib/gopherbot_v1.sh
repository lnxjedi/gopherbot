#!/bin/bash
# shellLib.sh - bash plugins should source this with 'source $GOPHER_INSTALLDIR/util/shellLib.sh'
# Return values for robot method calls
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
GBRET_RetryPrompt=14
GBRET_ReplyNotMatched=15
GBRET_UseDefaultValue=16
GBRET_TimeoutExpired=17
GBRET_Interrupted=18
GBRET_MatcherNotFound=19
GBRET_NoUserEmail=20
GBRET_NoBotEmail=21
GBRET_MailError=22
GBRET_InvalidPluginID=23
GBRET_UntrustedPlugin=24

# Plugin return values / exit codes, return values from CallPlugin
PLUGRET_Normal=0
PLUGRET_Fail=1
PLUGRET_MechanismFail=2
PLUGRET_ConfigurationError=3
PLUGRET_Success=7

base64_encode(){
	local MESSAGE
	MESSAGE=$(echo -n "$@" | base64)
	MESSAGE=$(echo -n "$MESSAGE")
	echo -n "$MESSAGE"
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
	"User": "$GOPHER_USER",
	"Channel": "$GOPHER_CHANNEL",
	"Format": "$GB_FORMAT",
	"Protocol": "$GOPHER_PROTOCOL",
	"PluginID": "$GOPHER_PLUGIN_ID",
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

gbExtract() {
	local JSON="$1"
	local ITEM="$2"
	echo "$JSON" | jq -r .$ITEM
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

CallPlugin(){
	local GB_FUNCNAME="CallPlugin"
	local PLUGNAME=$1
	shift
	local GB_FUNCARGS="{ \"PluginName\": \"$PLUGNAME\" }"
	local GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	local PLUGRETVAL=$(echo "$GB_RET" | jq .PlugRetVal)
	if [ "$PLUGRETVAL" -ne "$PLUGRET_Success" ]
	then
		return $PLUGRETVAL
	fi
	local PLUGPATH=$(echo "$GB_RET" | jq -r .PluginPath)
	local PLUGID=$(echo "$GB_RET" | jq -r .PluginID)
	GOPHER_PLUGIN_ID=$PLUGID $PLUGPATH "$@"
}

Remember(){
	if [ -z "$1" -o -z "$2" ]
	then
		return 1
	fi
	local GB_FUNCNAME="Remember"
	local R_KEY=$(base64_encode "$1")
	local R_MEMORY=$(base64_encode "$2")
	local GB_FUNCARGS=$(cat <<EOF
{
	"Key": "$R_KEY",
	"Value": "$R_MEMORY",
	"Base64" : true
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
	return 0
}

RememberContext(){
	if [ -z "$1" -o -z "$2" ]
	then
		return 1
	fi
	Remember "context:$1" "$2"
	return 0
}

Pause(){
	sleep $1
}

Recall(){
	if [ -z "$1" ]
	then
		return 1
	fi
	local R_KEY=$(base64_encode "$1")
	local GB_FUNCNAME="Recall"
	local GB_FUNCARGS=$(cat <<EOF
{
	"Key": "$R_KEY",
	"Base64": true
}
EOF
)
	local GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	local RETVAL=$(echo "$GB_RET" | jq -r .StrVal)
	echo -n "$RETVAL"
}

Elevate(){
	IMMEDIATE="false"
	if [ -n "$1" ]
	then
		IMMEDIATE = $1
	fi
	local GB_FUNCARGS=$(cat <<EOF
{
	"Immediate": "$IMMEDIATE"
}
EOF
)
	local GB_FUNCNAME="Elevate"
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
	gbExtract "$GB_RET" Attribute
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
	gbExtract "$GB_RET" Attribute
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
	gbExtract "$GB_RET" Attribute
	gbBotRet "$GB_RET"
}

Log(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="Log"
	local GLM_LEVEL="$1"
	local GLM_MESSAGE=$(base64_encode "$2")
	GB_FUNCARGS=$(cat <<EOF
{
	"Level": "$GLM_LEVEL",
	"Message": "$GLM_MESSAGE",
	"Base64" : true
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

PromptUserChannelForReply(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="PromptUserChannelForReply"
	local REGEX="$1"
	local PUSER="$2"
	local PCHANNEL="$3"
	local PROMPT=$(base64_encode "$4")
	GB_FUNCARGS=$(cat <<EOF
{
	"RegexID": "$REGEX",
	"User": "$PUSER",
	"Channel": "$PCHANNEL",
	"Prompt": "$PROMPT",
	"Base64" : true
}
EOF
)
	local RETVAL
	for TRY in 0 1 2
	do
		GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
		gbBotRet "$GB_RET"
		RETVAL=$?
		if [ $RETVAL -eq $GBRET_RetryPrompt ]
		then
			continue
		fi
		gbExtract "$GB_RET" Reply
		return $RETVAL
	done
	gbBotRet "$GB_RET"
	RETVAL=$?
	if [ $RETVAL -eq $GBRET_RetryPrompt ]
	then
		return $GBRET_Interrupted
	else
		return $RETVAL
	fi
}

PromptForReply(){
	local REGEX=$1
	shift
	PromptUserChannelForReply "$REGEX" "$GOPHER_USER" "$GOPHER_CHANNEL" "$*"
}

PromptUserForReply(){
	local REGEX=$1
	local PUSER=$2
	shift 2
	PromptUserChannelForReply "$REGEX" "$PUSER" "" "$*"
}

getFormat(){
	case "$1" in
	"-f")
		echo "Fixed"
		;;
	"-r")
		echo "Raw"
	esac
}

SendUserMessage(){
	if [[ $1 = -? ]]; then GB_FORMAT=$(getFormat $1); shift; else GB_FORMAT=variable; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendUserMessage"
	local SUM_USER=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(base64_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$SUM_USER",
	"Message": "$MESSAGE",
	"Base64" : true
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

SendUserChannelMessage(){
	if [[ $1 = -? ]]; then GB_FORMAT=$(getFormat $1); shift; else GB_FORMAT=variable; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendUserChannelMessage"
	local SUCM_USER=$1
	local SUCM_CHANNEL=$2
	shift 2
	MESSAGE="$*"
	MESSAGE=$(base64_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$SUCM_USER",
	"Channel": "$SUCM_CHANNEL",
	"Message": "$MESSAGE",
	"Base64" : true
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

SendChannelMessage(){
	if [[ $1 = -? ]]; then GB_FORMAT=$(getFormat $1); shift; else GB_FORMAT=variable; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendChannelMessage"
	local SCM_CHANNEL=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(base64_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"Channel": "$SCM_CHANNEL",
	"Message": "$MESSAGE",
	"Base64" : true
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

# Convenience functions so that copies of this logic don't wind up in a bunch of plugins
Say(){
	local FARG
	[[ $1 == -? ]] && { FARG=$1; shift; }
	if [ -n "$GOPHER_CHANNEL" ]
	then
		SendChannelMessage $FARG "$GOPHER_CHANNEL" "$*"
	else
		SendUserMessage $FARG "$GOPHER_USER" "$*"
	fi
}

Reply(){
	local FARG
	[[ $1 == -? ]] && { FARG=$1; shift; }
	if [ -n "$GOPHER_CHANNEL" ]
	then
		SendUserChannelMessage $FARG "$GOPHER_USER" "$GOPHER_CHANNEL" "$*"
	else
		SendUserMessage $FARG "$GOPHER_USER" "$*"
	fi
}
