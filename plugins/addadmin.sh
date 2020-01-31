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
- Keywords: [ "administrator" ]
  Helptext: [ "(bot), add admin <key> - add the user as a robot administrator" ]
CommandMatchers:
- Command: "add"
  Regex: '(?i:add ?admin(istrator)? ([^\s]+))'
ReplyMatchers:
- Label: "token"
  Regex: '.{8,}'
EOF
}

if [ "$command" == "configure" ]
then
    configure
    exit 0
fi

if [ "$command" == "init" ]
then
    exit 0
fi

checkReply(){
    if [ $1 -ne 0 ]
    then
        Say "Cancelling add admin"
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
    local FILE=${3:-conf/gopherbot.yaml}
    sed -i -e "s#$FIND#$REPLACE#g" "$GOPHER_CONFIGDIR/$FILE"
}

KEY=$1

if [ "$KEY" != "$SETUP_KEY" ]
then
    Say "Invalid setup key"
    exit 0
fi
SetParameter USER_KEY "$KEY"
Remember AUTH_USER "$GOPHER_USER"


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
mv conf/gopherbot.yaml conf/gopherbot.yaml.setup
mv conf/gopherbot-new.yaml conf/gopherbot.yaml
mv conf/plugins/builtin-admin.yaml conf/plugins/builtin-admin.yaml.setup
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
Say "Finally, encryption won't be initialized until the robot is restarted, so I'll go ahead and exit then restart ..."

AddCommand "builtin-admin" "quit"

exit 0

# Contents of new-robot.sh for mining; remove later!
#!/bin/bash -e

# new-robot.sh - set up a new robot and create
# the configuration repo.

GOPHER_INSTALL_DIR=$(dirname `readlink -f "$0"`)

REPO_DIR=$1
PROTOCOL=$2

usage(){
cat <<EOF
Usage: new-robot.sh <directory> <protocol>

Set up a new robot repository and perform initial configuration.
Protocol can be one of: slack, term.
EOF
    exit 1
}

if [ $# -ne 2 ]
then
    usage
fi

if ! mkdir -p "$REPO_DIR"
then
    echo "Unable to create destination directory"
    usage
fi
export GOPHER_SETUP_DIR=$(readlink -f $REPO_DIR)

cp -a $GOPHER_INSTALL_DIR/robot.skel/* $REPO_DIR
cp $GOPHER_INSTALL_DIR/robot.skel/.gitignore $REPO_DIR

cat <<EOF
Setting up new robot configuration repository in "$REPO_DIR".

This script will create a new robot directory and configure
you as a robot administrator, which provides access to
admin-only commands like 'reload', 'quit', 'update', etc.

The first part will prompt for required credentials, then
start the robot to complete setup using the 'setup' plugin.

EOF

GOPHER_PROTOCOL=slack

case $PROTOCOL in
slack)
    echo -n "Slack token? (from https://<org>.slack.com/services/new/bot) "
    read GOPHER_SLACK_TOKEN
    export GOPHER_SLACK_TOKEN
    ;;
term)
    export GOPHER_PROTOCOL=term
    export GOPHER_ADMIN=alice
    LOGFILE="/tmp/gopherbot-$REPO_DIR.log"
    GOPHER_ARGS="-l $LOGFILE"
    echo "Logging to $LOGFILE"
    ;;
esac

echo -n "Setup passphrase? (to be supplied to the robot) "
read GOPHER_SETUP_KEY
export GOPHER_SETUP_KEY

cat <<EOF
***********************************************************

Now we'll start the robot, which will start a connection
with the '${GOPHER_PROTOCOL}' protocol. Once it's connected,
open a private chat with your robot and tell it:

> setup $GOPHER_SETUP_KEY

(NOTE for 'term' protocol: use '|C' to switch to a private/
DM channel)

Press <enter>
EOF
read DUMMY

cd $REPO_DIR
ln -s $GOPHER_INSTALL_DIR/gopherbot .
./gopherbot $GOPHER_ARGS
# Start again after setup to reload and initialize encryption
[ -e "conf/gopherbot.yaml.setup" ] && ./gopherbot $GOPHER_ARGS
