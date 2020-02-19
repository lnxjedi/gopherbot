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
- Command: 'setup'
  Regex: '(?i:setup)'
MessageMatchers:
- Command: "setup"
  Regex: '(?i:^setup$)'
ReplyMatchers:
- Label: "alias"
  Regex: '[&!;:%#@~<>\/*+^\$?\\\[\]{}-]'
- Label: "encryptionkey"
  Regex: '[gG]|[^\s]{32,}'
- Label: "sshkey"
  Regex: '[gG]|[^\s]{16,}'
- Label: "token"
  Regex: '[gG]|[^\s]{8,}'
- Label: "repo"
  Regex: '[\w-_@:\/\\.]+'
- Label: "slacktoken"
  Regex: 'xoxb-[\w-]+'
- Label: "contquit"
  Regex: '(?i:c|q)'
EOF
}

if [ "$command" == "configure" ]
then
    configure
    exit 0
fi

ALIAS=$(GetBotAttribute "alias")
SLACKALIASES='!;-%~*+^\$?[]{}' # subset of all possible; others don't work

if [ "$command" == "init" ]
then
    NAME=$(GetBotAttribute "name")
    Pause 1
    if [ "$GOPHER_ENCRYPTION_INITIALIZED" ]
    then
        SendChannelMessage "general" "*******"
        SendChannelMessage "general" "Type '${ALIAS}setup' to continue setup..."
        exit 0
    fi
    SendChannelMessage "general" "*******"
    SendChannelMessage "general" "Welcome to the *Gopherbot* terminal connector. Since no \
configuration was detected, you're connected to '$NAME', the default robot."
    Pause 2
    SendChannelMessage "general" "If you've started the robot by mistake, just hit ctrl-D \
to exit and try 'gopherbot --help'; otherwise feel free to play around with the default robot - \
you can start by typing 'help'. If you'd like to start configuring a new robot, \
type: '${ALIAS}setup'."
    exit 0
fi

checkExit(){
    if [ $1 != 0 ]
    then
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
        if [ $RETVAL -eq 0 ]
        then
            echo "$REPLY"
            Say "Thanks"
            Pause 1
            return 0
        fi
        if [ -n "$TRY" ]
        then
            Say "Cancelling setup"
            return 1
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

getOrGenerate(){
    local REGEX=$1
    local SIZE=$2
    local BYTES=$(( $SIZE / 4 * 3 ))
    local PROMPT=$3
    for TRY in "" "" "LAST"
    do
        REPLY=$(PromptForReply "$REGEX" "$PROMPT")
        RETVAL=$?
        if [ $RETVAL -eq 0 ]
        then
            CHECK=$(echo $REPLY | tr [:upper:] [:lower:])
            if [ "$CHECK" == "g" ]
            then
                REPLY=$(dd status=none if=/dev/random bs=1 count=$BYTES | base64)
                echo "$REPLY"
                Say "Generated: $REPLY"
                Pause 1
                return 0
            else
                echo "$REPLY"
                Say "Thanks"
                Pause 1
                return 0
            fi
        fi
        if [ -n "$TRY" ]
        then
            Say "Cancelling setup"
            return 1
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

continueQuit(){
    RESP=$(getMatching 'contquit' "('c' to continue, 'q' to quit)")
    RET=$?
    RESP=$(echo "$RESP" | tr [:upper:] [:lower:])
    if [ $RET -ne 0 ] || [ "$RESP" != "c" ]
    then
        Say "Quitting setup, type '${ALIAS}setup' to start over"
        exit 0
    fi
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

if [ "$command" == "setup" -a -z "$GOPHER_ENCRYPTION_INITIALIZED" ]
then
    Say "Before we can get started, I need to initialize the encryption keys for \
your new robot."
    ENCRYPTION_KEY=$(getOrGenerate "encryptionkey" 32 \
    "Give me a string at least 32 characters long for the robot's encryption key (or 'g' to generate)")
    checkExit $?
    cat > .env <<EOF
GOPHER_ENCRYPTION_KEY=$ENCRYPTION_KEY
EOF
    Say "$(cat <<EOF
In just a moment I'll restart with encryption initialized; before restarting it'll help to have a few things ready to go:
1) You should have the credentials your robot will need for connecting to your team chat platform. For Slack, you should obtain a classic 'bot token' from the following URL:
https://<team>.slack.com/services/new/bot
2) You should have the clone URL for an empty git repository, accessible with ssh credentials, to hold your robot's configuration and state.
EOF
)"
    AddTask "restart-robot"
    exit 0
fi

if [ "$command" != "setup" ]
then
    exit 0
fi

Say "Welcome to the Gopherbot interactive setup plugin. I'll be asking a series of \
questions that I'll use to generate the initial configuration for your Gopherbot robot. \
At the end of the process, your robot can be pushed to a git repository."
# Get all the information
Say "First I'll need the Slack authentication token that your robot will use to connect to \
your team chat."
Pause 2
SLACK_TOKEN=$(getMatching "slacktoken" "Slack token?")
checkExit $?
Say "Don't worry - I'll encrypt that"

SLACK_TOKEN=${SLACK_TOKEN#xoxb-}
SLACK_ENCRYPTED=$($GOPHER_INSTALLDIR/gopherbot -l setup.log encrypt $SLACK_TOKEN)

Say "Next I'll need the name you'll use for your robot. To get your robot's attention \
it should be sufficient to start a command with e.g. a '@mention', but for maximum \
compatibility and portability to other chat platforms, your robot will always look for \
messages addressed to them; for example 'floyd, ping'."
Pause 2
BOTNAME=$(getMatching "SimpleString" "What do you want your robot's name to be?")
checkExit $?

BOTNAME=$(echo "$BOTNAME" | tr '[:upper:]' '[:lower:]')
CASENAME=$(echo "${BOTNAME:0:1}" | tr '[:lower:]' '[:upper:]')${BOTNAME:1}
BOTFULLNAME="$CASENAME Gopherbot"

Say "Now you can supply a one-character alias your robot will also recognize as it's \
name, chosen from this list: '$SLACKALIASES'. You'll probably use this most often \
for sending messages to your robot, as it's the most concise; e.g. ';ping'."
Pause 2
BOTALIAS=$(getMatching "alias" "Alias?")
checkExit $?
DISPALIAS="$BOTALIAS"
# '\' is an escape character and needs tons of special handling
if [[ $BOTALIAS = \\ ]]
then
    BOTALIAS='\\\\'
    DISPALIAS='\'
fi
Say "Your robot will likely run scheduled jobs periodically; for instance to back up \
it's long-term memories, or rotate a log file. Any output from these jobs will go to a \
default job channel for your robot. If you don't expect your robot to run a lot of jobs, \
it's safe to use e.g. 'general'. If you create a new channel for this purpose, be sure \
and invite your robot to the channel when it first connects."
Pause 2
JOBCHANNEL=$(getMatching "SimpleString" "Default job channel for your robot?")
checkExit $?

Say "I'll need an email address for your robot to use; it'll be used in the 'from:' when \
it sends email (configured separately), and also for git commits."
Pause 2
BOTMAIL=$(getMatching "Email" "Email address for your robot?")
checkExit $?

Say "Your robot will make heavy use of 'ssh' for doing it's work, and it's private keys \
will be encrypted. I'll need a passphrase your robot can use, at least 16 characters; don't \
worry if it's hard to type - the robot will get it right every time."
Pause 2
SSHPHRASE=$(getOrGenerate "sshkey" 16 "SSH Passphrase? \
(at least 16 characters, or 'g' to generate)")
checkExit $?
Say "I'll encrypt that, too; now I'll generate my ssh keypairs..."
continueQuit

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
checkExit $?

Pause 2
Say "$(cat <<EOF

In just a few moments I'll restart the Gopherbot daemon; when it starts, your new robot will connect to the team chat. At first it won't recognize you as an administrator, since your robot doesn't yet have access to the unique internal ID that your team chat assigns. I need to get a shared secret from you, so you can use the '${DISPALIAS}add admin xxxx' command.
EOF
)"
Pause 2
SETUPKEY=$(getOrGenerate "token" 8 "Shared secret (at least 8 chars, 'g' to generate)?")
checkExit $?

. .env
cat > .env <<EOF
GOPHER_ENCRYPTION_KEY=$GOPHER_ENCRYPTION_KEY
GOPHER_CUSTOM_REPOSITORY=$BOTREPO
# You should normally keep GOPHER_PROTOCOL commented out, except in
# production.
GOPHER_PROTOCOL=slack
# To use the deploy key below, add ssh/deploy_rsa.pub as a read-only
# deploy key for the custom configuration repository.
GOPHER_DEPLOY_KEY=$DEPKEY
EOF
Say "*******"
Say "$(cat <<EOF
I've created an '.env' file with environment variables you'll need for running your robot. It contains, among other things, the GOPHER_ENCRYPTION_KEY your robot will need to decrypt it's secrets, including it's credentials for team chat. With proper setup, it can also be used as-is to bootstrap your robot in a container.
This file should be kept in a safe place outside of a git repository - password managers are good for this. Here are it's contents:
EOF
)"
Pause 2
Say -f "$(echo; echo '--- snip ---'; cat .env; echo '--- /snip ---')"
continueQuit

Say "*******"
Say "$(cat <<EOF

For your robot to be able to save it's configuration, you'll need to configure a read-write deploy key in the repository settings. Here's the public-key portion (manage_rsa.pub) that you can paste in for your read-write deploy key:
EOF
)"
Pause 2
Say -f "$(echo; echo '--- snip ---'; cat custom/ssh/manage_rsa.pub; echo '--- /snip ---')"
continueQuit

Say "*******"
Say "$(cat <<EOF

While you're at it, you can also configure a read-only deploy key that corresponds to a flattened and unencrypted private key defined in 'GOPHER_DEPLOY_KEY', above. This deploy key will allow you to easily bootstrap your robot to a new container or VM, only requiring a few environment variables to be defined. Here's the public-key portion (deploy_rsa.pub) of the read-only deploy key:
EOF
)"
Say -f "$(echo; echo '--- snip ---'; cat custom/ssh/deploy_rsa.pub; echo '--- /snip ---')"
continueQuit

echo "GOPHER_SETUP_TOKEN=$SETUPKEY" >> .env

# Create configuration
cp -r $GOPHER_INSTALLDIR/robot.skel/* "$GOPHER_CONFIGDIR"
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
touch ".addadmin"
AddTask "restart-robot"
Pause 2
Say "Once you've recorded the '.env' and public deploy keys provided above, press <enter> to restart."
Pause 2
