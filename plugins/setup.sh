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
CommandMatchers:
- Command: "setup"
  Regex: '(?i:setup)'
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
  Regex: '[\w-_@:\/\\.]+'
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
    SendChannelMessage "general" "Hi, I'm $NAME, the default robot - I see you're running \
Gopherbot unconfigured."
    Pause 2
    SendChannelMessage "general" "If you've started the robot by mistake, just hit ctrl-D \
to exit and try 'gopherbot --help'; otherwise feel free to play around - \
you can start by typing 'help'. If you'd like to start configuring a new robot, \
type: '${ALIAS}setup'."
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
    local FILE=${3:-conf/gopherbot.yaml}
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

if [ "$command" != "setup" ]
then
    exit 0
fi

Say "Welcome to the Gopherbot interactive setup plugin. I'll be asking a series of \
questions that I'll use to generate the initial configuration for your Gopherbot robot. \
At the end of the process, the contents of the 'custom/' directory should be committed to \
a git repository."
# Get all the information
Say "First I'll need a Slack authentication token that your robot will use to connect to \
your Slack team. The best place to get a Gopherbot-compatible token is: \
https://<team>.slack.com/services/new/bot"
Pause 2
SLACK_TOKEN=$(getMatching "slacktoken" "Slack token?")
checkReply $?

SLACK_TOKEN=${SLACK_TOKEN#xoxb-}
SLACK_ENCRYPTED=$($GOPHER_INSTALLDIR/gopherbot -l setup.log encrypt $SLACK_TOKEN)

Say "Next I'll need the name you'll use for your robot. To get your robot's attention, \
it should be sufficient to use e.g. a '@mention', but for maximum compatibility and \
portability to other chat platforms, your robot will always look for messages addressed \
to them; for example 'floyd, ping'."
Pause 2
BOTNAME=$(getMatching "SimpleString" "What do you want your robot's name to be?")
checkReply $?

BOTNAME=$(echo "$BOTNAME" | tr '[:upper:]' '[:lower:]')
CASENAME=$(echo "${BOTNAME:0:1}" | tr '[:lower:]' '[:upper:]')${BOTNAME:1}
BOTFULLNAME="$CASENAME Gopherbot"

Say "Now you can supply a one-character alias your robot will also recognize as it's \
name, chosen from this list: '&!;:-%#@~<>/*+^\$?\[]{}'. You'll probably use this most often \
for sending messages to your robot, as it's the most concise; e.g. ';ping'."
Pause 2
BOTALIAS=$(getMatching "alias" "Alias?")
checkReply $?

Say "Your robot will likely run scheduled jobs periodically; for instance to back up \
it's long-term memories, or rotate a log file. Any output from these jobs will go to a
default job channel for your robot. If you don't expect your robot to run a lot of jobs,
it's safe to use e.g. 'general'."
Pause 2
JOBCHANNEL=$(getMatching "SimpleString" "Default job channel for your robot?")
checkReply $?

Say "I'll need an email address for your robot to use; it'll be used in the 'from:' when \
it sends email (configured separately), and also for git commits."
Pause 2
BOTMAIL=$(getMatching "Email" "Email address for your robot?")
checkReply $?

Say "Your robot will make heavy use of 'ssh' for doing it's work, and it's private keys \
will be encrypted. I'll need a passphrase your robot can use, at least 16 characters; don't \
worry if it's hard to type - the robot will get it right every time."
Pause 2
SSHPHRASE=$(getMatching "sshkey" "SSH Passphrase? \
(at least 16 characters)")
checkReply $?

SSH_ENCRYPTED=$($GOPHER_INSTALLDIR/gopherbot -l setup.log encrypt "$SSHPHRASE")
mkdir -p custom/ssh
ssh-keygen -N "$SSHPHRASE" -C "$BOTMAIL" -f custom/ssh/robot_rsa
ssh-keygen -N "$SSHPHRASE" -C "$BOTMAIL" -f custom/ssh/manage_rsa
ssh-keygen -N "" -C "$BOTMAIL" -f custom/ssh/deploy_rsa
DEPKEY=$(cat custom/ssh/deploy_rsa | tr ' \n' '_:')
rm -f custom/ssh/deploy_rsa

Say "After your robot has connected to your team chat for the first time, and you've \
added yourself as an administrator, you'll have the option of using the 'save' command to \
commit and push your robot's configuration to a git repository. The repository URL should \
be appropriate for ssh authentication."
Pause 2
BOTREPO=$(getMatching "repo" "Custom configuration repository URL?")
checkReply $?

Pause 2
Say "$(cat <<EOF

In just a few moments I'll restart the Gopherbot daemon; when it starts, YOUR new robot will connect to the team chat. At first it won't recognize you as an administrator, since your robot doesn't yet have access to the unique internal ID that your team chat assigns. I need to get a shared secret from you, so you can use the '${BOTALIAS}add admin xxxx' command.
EOF
)"
Pause 2
SETUPKEY=$(getMatching "token" "Shared secret (at least 8 chars)?")
checkReply $?

. .env
cat > .env <<EOF
GOPHER_ENCRYPTION_KEY=$GOPHER_ENCRYPTION_KEY
GOPHER_CUSTOM_REPOSITORY=$BOTREPO
# To use the deploy key below, add ssh/deploy_rsa.pub as a read-only deploy key
# for the custom configuration repository.
GOPHER_DEPLOY_KEY=$DEPKEY
EOF
Say "$(cat <<EOF
I've created an '.env' file with environment variables you'll need for running your robot. It contains, among other things, the GOPHER_ENCRYPTION_KEY your robot will need to decrypt it's secrets, including it's credentials for team chat. With proper setup, it can also be used as-is to bootstrap your robot in a container.
This file should be kept in a safe place outside of a git repository - password managers are good for this. Here are it's contents:
EOF
)"
Pause 2
Say -f "$(echo; cat .env)"

Say "$(cat <<EOF

For your robot to be able to save it's configuration, you'll need to configure a read-write deploy key in the repository settings. Here's the public-key portion that you can paste in for your read-write deploy key:
EOF
)"
Pause 2
Say -f "$(echo; cat custom/ssh/manage_rsa.pub)"

Pause 2
Say "$(cat <<EOF

While you're at it, you can also configure a read-only deploy key that corresponds to a flattened and unencrypted private key defined in 'GOPHER_DEPLOY_KEY', above. This deploy key will allow you to easily bootstrap your robot to a new container or VM, only requiring a few environment variables to be defined. Here's the public-key portion of the read-only deploy key:
EOF
)"
Say -f "$(echo; cat custom/ssh/deploy_rsa.pub)"

echo "GOPHER_SETUP_TOKEN=$SETUPKEY" >> .env

# Create configuration
cp -a $GOPHER_INSTALLDIR/robot.skel/* "$GOPHER_CONFIGDIR"
substitute "<defaultprotocol>" "slack" # slack-only for now
substitute "<slackencrypted>" "$SLACK_ENCRYPTED" "conf/slack.yaml"
substitute "<sshencrypted>" "$SSH_ENCRYPTED"
substitute "<jobchannel>" "$JOBCHANNEL" "conf/slack.yaml"
substitute "<botname>" "$BOTNAME"
substitute "<botalias>" "$BOTALIAS"
substitute "<botfullname>" "$BOTFULLNAME"
substitute "<botfullname>" "$BOTFULLNAME" "git/config"
substitute "<botemail>" "$BOTMAIL"
substitute "<botemail>" "$BOTMAIL" "git/config"
Pause 2
Say "I've created the initial configuration for your robot, and now I'm ready to \
restart the daemon. Once your robot has connected, you'll need to identify yourself with \
a private message: \
'add administrator $SETUPKEY'."
AddTask "restart-robot"
Pause 2
Say "Once you've recorded the '.env' and public deploy keys provided above, press <enter> to restart."
Pause 2
