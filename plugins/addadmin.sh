#!/bin/bash

# addadmin.sh - plugin for adding an administrator to a new robot

# START Boilerplate
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift
# END Boilerplate

configure(){
	cat <<"EOF"
---
AllChannels: true
AllowDirect: true
Help:
- Keywords: [ "administrator", "add" ]
  Helptext: [ "(bot), add admin <key> - add the user as a robot administrator" ]
CommandMatchers:
- Command: "add"
  Regex: '(?i:add ?admin(?:istrator)? ([^\s]+))'
EOF
}

if [ "$command" == "configure" ]
then
    configure
    exit 0
fi

if [ "$command" != "add" ]
then
    exit 0
fi

checkReply(){
    if [ $1 -ne 0 ]
    then
        Say "Cancelling setup"
        exit 0
    fi
    Say "Thanks"
    Pause 1
}

getMatching(){
    local REGEX=$1
    local PROMPT=$2
    for TRY in "" "" "LAST"
    do
        REPLY=$(PromptForReply "$REGEX" "$PROMPT")
        RETVAL=$?
        [ $RETVAL -eq 0 ] && { echo "$REPLY"; return 0; }
        if [ -n "$TRY" ]
        then
            return $RETVAL
        fi
        case $RETVAL in
        $GBRET_ReplyNotMatched)
            Say "Try again? Your answer doesn't match the pattern for $REGEX"
            ;;
        $GBRET_TimeoutExpired)
            Say "Try again? Timeout expired waiting for your reply"
            ;;
        *)
            return $RETVAL
        esac
    done
}

substitute(){
    local FIND=$1
    local REPLACE=$2
    local FILE=${3:-conf/robot.yaml}
    REPLACE=${REPLACE//\\/\\\\}
    for TRY in "#" "|" "%" "^"
    do
        if [[ ! $REPLACE = *$TRY* ]]
        then
            RC="$TRY"
            break
        fi
    done
    sed -i -e "s${RC}$FIND${RC}$REPLACE${RC}g" "$GOPHER_CONFIGDIR/$FILE"
}

KEY=$1

if [ "$KEY" != "$GOPHER_SETUP_TOKEN" ]
then
    Say "Invalid shared secret; check your GOPHER_SETUP_TOKEN and try again."
    exit 0
fi
if [ ! -e ".addadmin" ]
then
    Say "Setup token already used!!"
    exit 1
fi
rm -f ".addadmin"
sed -i ".env" -e '/^GOPHER_SETUP_TOKEN=/d'
USERID=$(GetSenderAttribute "id")
USERID=${USERID#<}
USERID=${USERID%>}
USERNAME=$(getMatching "SimpleString" \
    "What username do you want the robot to know you by?")
checkReply $?
Say "Detected User ID $USERID for $USERNAME"
substitute "<adminusername>" "$USERNAME" "conf/slack.yaml"
substitute "<adminuserid>" "$USERID" "conf/slack.yaml"

AddTask "restart-robot"
AddTask say "You've been successfully added as an administrator. \
If you've configured 'manage_key.pub' as a read-write deploy key \
for my repository, you can use the 'save' command to upload my \
configuration. Have fun."
