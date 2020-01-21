#!/bin/bash

# setup.sh - plugin for setting up a new robot

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
  Helptext: [ "(bot), setup - perform initial setup of a new robot" ]
- Keywords: [ "administrator" ]
  Helptext: [ "(bot), add admin <key> - add the user as a robot administrator" ]
CommandMatchers:
- Command: "setup"
  Regex: '(?i:setup)'
- Command: "add"
  Regex: '(?i:add ?admin(istrator)? ([^\s]+))'
MessageMatchers:
- Command: "setup"
  Regex: '(?i:setup)'
ReplyMatchers:
- Label: "alias"
  Regex: '[&!;:%#@~<>\/*+^\$?\\\[\]{}-]'
- Label: "encryptionkey"
  Regex: '.{32,}'
- Label: "sshkey"
  Regex: '.{16,}'
- Label: "token"
  Regex: '.{8,}'
- Label: "repo"
  Regex: "[\w-_@:/\\]+"
- Label: "slacktoken"
  Regex: 'xoxb-[\w-]+'
EOF
}

if [ "$command" == "configure" ]
then
    configure
    exit 0
fi

if [ "$command" == "init" -a "$GOPHER_UNCONFIGURED" ]
then
    NAME=$(GetBotAttribute "name")
    ALIAS=$(GetBotAttribute "alias")
    Pause 1
    if [ "$GOPHER_ENCRYPTION_INITIALIZED" ]
    then
        SendChannelMessage "general" "Type '${ALIAS}setup' to continue setup..."
        exit 0
    fi
    SendChannelMessage "general" "*******"
    SendChannelMessage "general" "Hi, I'm $NAME, the default robot - I see you're running Gopherbot unconfigured"
    Pause 2
    SendChannelMessage "general" "If you've started the robot by mistake, just hit ctrl-D to exit and try \
'gopherbot --help'; otherwise feel free to play around with Gopherbot; you can start by typing 'help'. \
If you'd like to start configuring a new robot, type: '${ALIAS}setup'."
    exit 0
fi

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
    local FILE=${3:-conf/gopherbot.yaml}
    sed -i -e "s#$FIND#$REPLACE#g" "$GOPHER_CONFIGDIR/$FILE"
}

if [ "$command" == "setup" -a -z "$GOPHER_ENCRYPTION_INITIALIZED" ]
then
    Say "Before we can get started, we need to set up encryption"
    ENCRYPTION_KEY=$(getMatching "encryptionkey" \
    "Give me a string at least 32 characters long for the robot's encryption key")
    checkReply $?
    cat > .env <<EOF
GOPHER_ENCRYPTION_KEY=$ENCRYPTION_KEY
EOF
    Say "Now I'll restart with encryption initialized..."
    AddTask "restart-robot"
    exit 0
fi

if [ "$command" == "setup" ]
then
    Say "Welcome to the Gopherbot interactive setup plugin. I'll be asking a series of \
questions that I'll use to generate the initial configuration for your Gopherbot robot. \
At the end of the process, the contents of the 'custom/' directory should be committed to \
a git repository."
    # Get all the information
    SLACK_TOKEN=$(getMatching "slacktoken" "Slack token? (from https://<org>.slack.com/services/new/bot)")
    SLACK_TOKEN=${SLACK_TOKEN#xoxb-}
    SLACK_ENCRYPTED=$($GOPHER_INSTALLDIR/gopherbot -l setup.log encrypt $SLACK_TOKEN)
    checkReply $?
    BOTNAME=$(getMatching "SimpleString" "What do you want your robot's name to be?")
    checkReply $?
    BOTNAME=$(echo "$BOTNAME" | tr '[:upper:]' '[:lower:]')
    CASENAME=$(echo "${BOTNAME:0:1}" | tr '[:lower:]' '[:upper:]')${BOTNAME:1}
    BOTFULLNAME="$CASENAME Gopherbot"
    BOTALIAS=$(getMatching "alias" \
    "Pick a one-character alias for your robot from '&!;:-%#@~<>/*+^\$?\[]{}'")
    checkReply $?
    BOTMAIL=$(getMatching "Email" "Email address for the robot? (will be used in git commits)")
    checkReply $?
    SETUPKEY=$(getMatching "token" "I need a one-time password from you; you'll use this to \
identify and add yourself as an administrator after I connect to team chat.\n\nPassword? \
(at least 8 chars)")
    checkReply $?
    echo "GOPHER_SETUP_TOKEN=$SETUPKEY" >> .env
    SSHPHRASE=$(getMatching "sshkey" "Passphrase to use for encrypting my ssh private key? \
(at least 16 characters)")
    checkReply $?
    SSH_ENCRYPTED=$($GOPHER_INSTALLDIR/gopherbot -l setup.log encrypt "$SSHPHRASE")
    # Create configuration
    cp -a $GOPHER_INSTALLDIR/robot.skel/* "$GOPHER_CONFIGDIR"
    Say "Generating my ssh keypair..."
    ssh-keygen -N "$SSHPHRASE" -C "$BOTMAIL" -f custom/ssh/robot_rsa
    Say "Here's my public key, which you can use to grant write access to my configuration \
and state repositories:"
    Say -f "$(cat custom/ssh/robot_rsa.pub)"
    CUSTOM_REPO=$(getMatching "repo" "What's the URL to use for my custom configuration \
repository? If you're going to save my configuration with the 'save' adminstrative \
command, this needs to be an ssh url that I can push with my private key.\n\nRepository URL?")
    substitute "<defaultprotocol>" "slack" # slack-only for now
    substitute "<slackencrypted>" "$SLACK_ENCRYPTED" "conf/slack.yaml"
    substitute "<sshencrypted>" "$SSH_ENCRYPTED"
    substitute "<botname>" "$BOTNAME"
    substitute "<botalias>" "$BOTALIAS"
    substitute "<botfullname>" "$BOTFULLNAME"
    substitute "<botfullname>" "$BOTFULLNAME" "git/config"
    substitute "<botemail>" "$BOTMAIL"
    substitute "<botemail>" "$BOTMAIL" "git/config"
    Pause 3
    Say "Initial configuration is nearly complete. Before we can configure you as an \
administrator, I'll need to restart and connect to your team chat with the supplied \
credentials. Once connected, you'll need to identify yourself with a private message: \
\"add administrator $SETUPKEY\""
    Say "Once you've been configured as an administrator, you can use the 'save' command \
to save my initial configuration if my repository has been configured for push. Otherwise \
you should commit and push the contents of 'custom/' manually."
    AddTask "restart-robot"
    exit 0
fi

# Running again as a pipeline task
if [ "$command" == "continue" ]
then
    Pause 2
    
fi

exit 0
if [ "$command" != "setup" ]
then
    exit 0
fi

exit 0
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
