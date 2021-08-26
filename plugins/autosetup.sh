#!/bin/bash -e

# setup.sh - plugin for setting up a new robot

command=$1
shift

if [ "$command" == "configure" ]
then
cat << "EOF"
Help:
- Keywords: [ "setup" ]
  Helptext: [ "(bot), setup <protocol> - display the answerfile for the given protocol" ]
CommandMatchers:
- Command: 'setup'
  Regex: '(?i:setup (\w+))'
MessageMatchers:
- Command: "setup"
  Regex: '(?i:^setup (\w+)$)'
EOF
    exit 0
fi

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}" >&2
    # The script should usually exit on error
    exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

for REQUIRED in git jq ssh
do
    if ! which $REQUIRED >/dev/null 2>&1
    then
        echo "Required '$REQUIRED' not found in \$PATH" >&2
        exit 1
    fi
done

# START Boilerplate
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh
# END Boilerplate

if [ "$command" == "setup" ]
then
    PROTOCOL=$1
    ANSFILE="$GOPHER_INSTALLDIR/resources/answerfiles/$PROTOCOL.txt"
    if [ ! -e "$ANSFILE" ]
    then
        Say "Protocol answerfile template not found: $ANSFILE"
        exit 0
    fi
    if [ ! "$GOPHER_CONTAINER" ]
    then
        if [ -e "answerfile.txt" ]
        then
            Say "Not over-writing existing 'answerfile.txt'"
            exit 0
        fi
        cp "$ANSFILE" "answerfile.txt"
        if [ ! -e "gopherbot" ]
        then
            ln -s "$GOPHER_INSTALLDIR/gopherbot" .
        fi
        Say "Edit 'answerfile.txt' and re-run gopherbot with no arguments to generate your robot."
        FinalTask robot-quit
        exit 0
    fi
    # Running in a container
    ANSTXT="$(cat $ANSFILE)"
    Say -f "$(cat <<EOF
Copy to answerfile.txt:
<-- snip answerfile.txt -->
$ANSTXT
<-- /snip -->

Edit your 'answerfile.txt' and run the container with '--env-file answerfile.txt'.
EOF
)"
    FinalTask robot-quit
    exit 0
fi

if [ "$command" != "init" ]
then
    exit 0
fi

if [ -e "answerfile.txt" ]
then
    source "answerfile.txt"
elif [ ! "$ANS_PROTOCOL" ]
then
    exit 0
fi

# checkExit VAR [regex] [g]
checkExit() {
    local VALUE="${!1}"
    if [ ! "$VALUE" ]
    then
        Say "Missing value for \"$1\", quitting..."
        AddTask robot-quit
        exit 0
    fi
    if [ "$2" ]
    then
        if [ "$VALUE" == "$3" ]
        then
            return
        fi
        if ! echo "$VALUE" | grep -qE "$2"
        then
            Say "Value \"$VALUE\" doesn't match regex \"$2\", quitting..."
            AddTask robot-quit
            exit 0
        fi
    fi
}

getOrGenerate(){
    local VARNAME=$1
    local SIZE=$2

    local VALUE=${!VARNAME}
    local BYTES=$(( $SIZE / 4 * 3 ))

    VALUE=${!1}
    if [ "$VALUE" == "g" ]
    then
        VALUE=$(dd status=none if=/dev/random bs=1 count=$BYTES | base64)
    fi
    echo "$VALUE"
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

if [ ! "$GOPHER_ENCRYPTION_INITIALIZED" ]
then
    checkExit "ANS_ENCRYPTION_KEY"
    Say "Initializing encryption and restarting..."
    #set -x
    ENCRYPTION_KEY=$(getOrGenerate ANS_ENCRYPTION_KEY 32)
    cat > .env <<EOF
GOPHER_ENCRYPTION_KEY=$ENCRYPTION_KEY
EOF
    AddTask restart-robot
    exit 0
fi

Say "Continuing automatic setup..."

checkExit "ANS_SLACK_TOKEN" '^xoxb-[0-9A-Za-z-]+$'
checkExit "ANS_ROBOT_NAME" '^[0-9A-Za-z_-]+$'
checkExit "ANS_ROBOT_ALIAS" '^[]!;%~*+^$?[\{\}-]$'
checkExit "ANS_JOB_CHANNEL" '^[0-9A-Za-z_-]+$'
checkExit "ANS_ROBOT_EMAIL" '^[0-9A-Za-z+\.\_\-]*@[0-9A-Za-z+\.\_\-]*$'
checkExit "ANS_SSH_PHRASE" '^[0-9A-Za-z_+/-]{16,}$' "g"
checkExit "ANS_KEY_TYPE" '^dsa|ecdsa|rsa|ed25519$'
checkExit "ANS_ROBOT_REPOSITORY"
checkExit "ANS_ADMIN_SECRET" '^[0-9A-Za-z_+/-]{8,}$' "g"

SLACK_TOKEN=${ANS_SLACK_TOKEN#xoxb-}
SLACK_ENCRYPTED=$($GOPHER_INSTALLDIR/gopherbot encrypt $SLACK_TOKEN)
BOTNAME=$(echo "$ANS_ROBOT_NAME" | tr '[:upper:]' '[:lower:]')
CASENAME=$(echo "${BOTNAME:0:1}" | tr '[:lower:]' '[:upper:]')${BOTNAME:1}
BOTFULLNAME="$CASENAME Gopherbot"

BOTALIAS="$ANS_ROBOT_ALIAS"
DISPALIAS="$BOTALIAS"
# '\' is an escape character and needs tons of special handling
if [[ $BOTALIAS = \\ ]]
then
    BOTALIAS='\\\\'
    DISPALIAS='\'
fi

JOBCHANNEL="$ANS_JOB_CHANNEL"
BOTMAIL="$ANS_ROBOT_EMAIL"

KEY_TYPE=${ANS_KEY_TYPE:-rsa}

SSHPHRASE="$(getOrGenerate ANS_SSH_PHRASE 16)"
Say "Generating ssh keys..."
sleep 1
SSH_ENCRYPTED=$($GOPHER_INSTALLDIR/gopherbot encrypt "$SSHPHRASE")
mkdir -p custom/ssh
ssh-keygen -N "$SSHPHRASE" -C "$BOTMAIL" -t "$KEY_TYPE" -f custom/ssh/robot_key
ssh-keygen -N "$SSHPHRASE" -C "$BOTMAIL" -t "$KEY_TYPE" -f custom/ssh/manage_key
ssh-keygen -N "" -C "$BOTMAIL" -t "$KEY_TYPE" -f custom/ssh/deploy_key
DEPKEY=$(cat custom/ssh/deploy_key | tr ' \n' '_:')
rm -f custom/ssh/deploy_key

BOTREPO="$ANS_ROBOT_REPOSITORY"
SETUPKEY="$(getOrGenerate ANS_ADMIN_SECRET 8)"

source .env
cat > .env <<EOF
GOPHER_ENCRYPTION_KEY=$GOPHER_ENCRYPTION_KEY
GOPHER_CUSTOM_REPOSITORY=$BOTREPO
## You should normally keep GOPHER_PROTOCOL commented out, except when
## used in a production container. This allows for the normal case where
## the robot starts in terminal mode for local development.
GOPHER_PROTOCOL=slack
## To use the deploy key below, add ssh/deploy_key.pub as a read-only
## deploy key for the custom configuration repository.
GOPHER_DEPLOY_KEY=$DEPKEY
GOPHER_SETUP_TOKEN=$SETUPKEY
EOF

# Create configuration
cp -r $GOPHER_INSTALLDIR/robot.skel/* "$GOPHER_CONFIGDIR"
substitute "<slackencrypted>" "$SLACK_ENCRYPTED" "conf/slack.yaml"
substitute "<sshencrypted>" "$SSH_ENCRYPTED"
substitute "<jobchannel>" "$JOBCHANNEL" "conf/slack.yaml"
substitute "<botname>" "$BOTNAME"
substitute "<botalias>" "$BOTALIAS"
substitute "<botalias>" "$BOTALIAS" "conf/terminal.yaml"
substitute "<botfullname>" "$BOTFULLNAME"
substitute "<botfullname>" "$BOTFULLNAME" "git/config"
substitute "<botemail>" "$BOTMAIL"
substitute "<botemail>" "$BOTMAIL" "git/config"

touch ".addadmin"
if [ ! -e "gopherbot" ]
then
    ln -s "$GOPHER_INSTALLDIR/gopherbot" .
fi
echo
echo
if [ "$GOPHER_CONTAINER" ]
then
    echo "Generated files (between <-- snip ...>/<-- /snip --> lines):"
    echo
    cat <<EOF
<-- snip .env -->
GOPHER_ENCRYPTION_KEY=$GOPHER_ENCRYPTION_KEY
GOPHER_CUSTOM_REPOSITORY=$BOTREPO
## You should normally keep GOPHER_PROTOCOL commented out, except when
## used in a production container. This allows for the normal case where
## the robot starts in terminal mode for local development.
# GOPHER_PROTOCOL=slack
## To use the deploy key below, add ssh/deploy_key.pub as a read-only
## deploy key for the custom configuration repository.
GOPHER_DEPLOY_KEY=$DEPKEY
<-- /snip >
EOF
    echo
    echo "<-- snip manage_key.pub -->"
    cat "custom/ssh/manage_key.pub"
    echo "<-- /snip >"
    echo
    echo "<-- snip deploy_key.pub -->"
    cat "custom/ssh/deploy_key.pub"
    echo "<-- /snip >"
    echo
    Say "********************************************************

"
    Say "Initial configuration of your robot is complete. To finish setting up your robot, \
and to add yourself as an administrator:
1) Add a read-write deploy key to the robot's repository, using the the 'manage_key.pub' \
shown above; this corresponds to an encrypted 'manage_key' that your robot will use to save \
and update it's configuration. 
2) Add a read-only deploy key to the robot's repository, using the 'deploy_key.pub' shown \
above; this corresponds to an unencrypted 'deploy_key' (file removed) which is trivially \
encoded as the 'GOPHER_DEPLOY_KEY' in the '.env' file. *Gopherbot* will use this deploy key, \
along with the 'GOPHER_CUSTOM_REPOSITORY', to initially clone it's repository during bootstrapping.
3) Copy the contents of the '.env' file shown above to a safe place, not kept in a repository. \
GOPHER_PROTOCOL is commented out to avoid accidentally connecting another instance of your robot \
to team chat when run from a terminal window for the development environment. With proper \
configuration of your git repository, the '.env' file is all that's needed to bootstrap your \
robot in to an empty *Gopherbot* container, (https://quay.io/lnxjedi/gopherbot) or on a Linux \
host or VM with the *Gopherbot* software archive installed.
4) Once these tasks are complete, re-start this container in a separate tab/window to connect \
your robot to team chat.
5) Invite your robot to #${JOBCHANNEL}; slack robots will need to be invited to any channels \
where they will be listening and/or speaking.
6) Open a direct message (DM) channel to your robot, and give this command to add yourself \
as an administrator: \"add administrator $SETUPKEY\"; your robot will then update \
'custom/conf/slack.yaml' to make you an administrator, and reload it's configuration.
7) Once that completes, you can instruct the robot to store it's configuration in it's git \
repository by issuing the 'save' command.
8) At this point, feel free to experiment with the default robot; you can start by typing \
\"help\" in ${JOBCHANNEL}. When you're finished, press <ctrl-c> in the window where you \
ran \"gopherbot\" to stop the robot, or optionally tell your robot to \"${BOTALIAS}quit\".

After your robot has saved it's configuration, you can stop and discard this container."

    AddTask robot-quit
    exit 0
else
    Say "********************************************************

"
    Say "Initial configuration of your robot is complete. To finish setting up your robot, \
and to add yourself as an administrator:
1) Open a second terminal window in the same directory as answerfile.txt; you'll need this \
for completing setup.
2) Add a read-write deploy key to the robot's repository, using the contents of \
'custom/ssh/manage_key.pub'; this corresponds to an encrypted 'manage_key' that your \
robot will use to save and update it's configuration.
3) Add a read-only deploy key to the robot's repository, using the contents of \
'custom/ssh/deploy_key.pub'; this corresponds to an unencrypted 'deploy_key' (file removed) \
which is trivially encoded as the 'GOPHER_DEPLOY_KEY' in the '.env' file. *Gopherbot* will \
use this deploy key, along with the 'GOPHER_CUSTOM_REPOSITORY', to initially clone it's \
repository during bootstrapping.
4) Once these tasks are complete, in your second terminal window, run './gopherbot' without any \
arguments. Your robot should connect to your team chat.
5) Invite your robot to #${JOBCHANNEL}; slack robots will need to be invited to any channels \
where they will be listening and/or speaking.
6) Open a direct message (DM) channel to your robot, and give this command to add yourself \
as an administrator: \"add administrator $SETUPKEY\"; your robot will then update \
'custom/conf/slack.yaml' to make you an administrator, and reload it's configuration.
7) Once that completes, you can instruct the robot to store it's configuration in it's git \
repository by issuing the 'save' command.
8) At this point, feel free to experiment with the default robot; you can start by typing \
\"help\" in ${JOBCHANNEL}. When you're finished, press <ctrl-c> in the window where you \
ran \"gopherbot\" to stop the robot, or optionally tell your robot to \"${BOTALIAS}quit\".
9) Finally, copy the contents of the '.env' file to a safe place, with the GOPHER_PROTOCOL \
commented out; this avoids accidentally connecting another instance of your robot to team chat \
when run from a terminal window for the development environment. With proper configuration of \
your git repository, the '.env' file is all that's needed to bootstrap your robot in to an empty \
*Gopherbot* container, (https://quay.io/lnxjedi/gopherbot) or on a Linux host or VM with the \
*Gopherbot* software archive installed.

Now you've completed all of the initial setup for your *Gopherbot* robot. See the chapter on \
deploying and running your robot (https://lnxjedi.github.io/gopherbot/RunRobot.html) for \
information on day-to-day operations. You can stop the running robot in your second terminal \
window using <ctrl-c>.

(NOTE: Scroll back to the line of *** above and follow the directions to finish setup)"

    AddTask robot-quit
    exit 0
fi
