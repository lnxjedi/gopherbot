#!/bin/bash

# echo.sh - trivial shell plugin example for Gopherbot

# START Boilerplate
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift
# END Boilerplate

configure(){
	cat <<"EOF"
---
Help:
- Keywords: [ "setup" ]
  Helptext: [ "(bot), setup <key> - perform initial setup of your robot" ]
CommandMatchers:
- Command: "setup"
  Regex: '(?i:setup ([^\s]+))'
ReplyMatchers:
- Label: "alias"
  Regex: '([&!;:%#@~<>\/*+^\$?\\\[\]{}-])'
- Label: "encryptionkey"
  Regex: '(.{32,})'
EOF
}

if [ "$command" == "configure" ]
then
    configure
    exit 0
fi

if [ "$command" != "setup" ]
then
    exit 0
fi

KEY=$1

if [ "$KEY" != "$SETUP_KEY" ]
then
    Say "Invalid setup key"
    exit 0
fi
SetParameter USER_KEY "$KEY"

checkReply(){
    if [ $1 -ne 0 ]
    then
        Say "Cancelling setup"
        exit 0
    fi
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
    sed -i -e "s/$FIND/$REPLACE/g" conf/gopherbot.yaml
}

USERNAME=$(GetSenderAttribute "user")
USERID=$(GetSenderAttribute "id")
USERID=${USERID#<}
USERID=${USERID%>}
if [ -z "$USERNAME" ]
then
    USERNAME=$(getMatching "SimpleString" \
      "What username do you want the robot to know you by?")
fi
Say "Detected User ID $USERID for $USERNAME"
BOTNAME=$(getMatching "SimpleString" \
  "What do you want your robot's name to be?")
checkReply $?
BOTALIAS=$(getMatching "alias" \
  "Pick a one-character alias for your robot from '&!;:-%#@~<>/*+^\$?\[]{}'")
checkReply $?
ENCRYPTION_KEY=$(getMatching "encryptionkey" \
  "Give me a string at least 32 characters long for the robot's encryption key")
checkReply $?
cat > .env <<EOF
GOPHER_SLACK_TOKEN=$SLACK_TOKEN
GOPHER_ENCRYPTION_KEY=$ENCRYPTION_KEY
EOF
mv conf/gopherbot.yaml conf/gopherbot.yaml.setup
mv conf/gopherbot.yaml.new conf/gopherbot.yaml
substitute "<GOPHER_ADMIN_USER>" "$USERNAME"
substitute "<GOPHER_ADMIN_ID>" "$USERID"
substitute "<GOPHER_BOTNAME>" "$BOTNAME"
substitute "<GOPHER_ALIAS>" "$BOTALIAS"
git init
git add .
git commit -m "Initial commit from Gopherbot setup"
Say "Initial setup complete - the configuration repository in $(pwd) is ready for 'git remote add origin ...; git push'"
Pause 3
Say "The contents of $(pwd)/.env need to be preserved separately, as it contains secrets and is excluded in .gitignore"
Pause 3
Say "Finally, encryption won't be initialized until the robot is restarted, but I'll go ahead and reload my configuration..."

AddCommand "builtin-admin" "reload"
