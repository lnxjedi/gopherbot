#!/bin/bash -e

# remote-exec.sh - script for the `remote` task, for running simple commands
# on a remote system. To use the remote task:
# 1) Add the `ssh-init` task to start the robot's ssh agent
# 2) set the following environment vars for the pipeline with
#    `SetParameter <name> <value>`:
# - GOPHER_REMOTE_HOST - required if (-h <host>) not given
# - GOPHER_REMOTE_USER - optional, override with (-l <loginid>); defaults to $USER
# - GOPHER_REMOTE_DIR - optional remote directory, override with (-d <dir>);
#   defaults to remote $HOME

# Technical note: this task 

# Usage: `AddTask remote (-A) (-l <login>) (-h <host>) (-f <file>|-s <scanhost(:port)>|<remote command>)`
#   Executes a command on a remote host. Passing `-A` forwards the robot's ssh agent.
#
#   If `-f <file>` is given, <file> is executed remotely and any further arguments
#   are ignored. NOTE: When "-f <file>" is given, this script prepends "set -e" to the
#   remote script, which causes a non-zero exit on failed commands.
#
#   If `-s <scanhost(:port)` is given, ssh-keyscan is run from the remote
#   host to add a host to known_hosts on the remote system; this should be done ahead
#   of any commands that use ssh remotely. Further arguments are ignored when "-s" is
#   given.
#

unset GR_FORWARD_AGENT GR_REMOTE_HOST GR_REMOTE_SCANHOST GR_REMOTE_SCRIPT GR_REMOTE_USER GR_REMOTE_DIR

[ -n "$GOPHER_REMOTE_USER" ] && GR_REMOTE_USER="$GOPHER_REMOTE_USER"
[ -n "$GOPHER_REMOTE_HOST" ] && GR_REMOTE_HOST="$GOPHER_REMOTE_HOST"
[ -n "$GOPHER_REMOTE_DIR" ] && GR_REMOTE_DIR="$GOPHER_REMOTE_DIR"

while getopts ":Al:h:f:s:d:" OPT
do
    case $OPT in
    A)
        GR_FORWARD_AGENT="-A"
        ;;
    l)
        GR_REMOTE_USER="-l $OPTARG"
        ;;
    h)
        GR_REMOTE_HOST="$OPTARG"
        ;;
    f)
        GR_REMOTE_SCRIPT="$OPTARG"
        ;;
    s)
        GR_REMOTE_SCANHOST="$OPTARG"
        ;;
    d)
        GR_REMOTE_DIR="$OPTARG"
        ;;
    \?)
        echo "Invalid option: $OPTARG" >&2
        exit 1
        ;;
    : )
        echo "Invalid option: $OPTARG requires an argument" >&2
        exit 1
        ;;
    esac
done
shift $((OPTIND -1))

# spit out the remote script with "set -e" prepended
remote_script(){
    echo "set -e"
    if [ -n "$GR_REMOTE_DIR" ]
    then
        echo "cd $GR_REMOTE_DIR"
    fi
    cat "$GR_REMOTE_SCRIPT"
}

# spit out a remote script for ssh-keyscan on a remote host
remote_scan(){
    REMOTE_HOST=$GR_REMOTE_SCANHOST
    SCAN_HOST=$REMOTE_HOST
    local REMOTE_PORT
    if [[ $REMOTE_HOST = *:* ]]
    then
        REMOTE_PORT=${REMOTE_HOST##*:}
        REMOTE_HOST=${REMOTE_HOST%%:*}
        SKARGS="-p $REMOTE_PORT"
    fi
    cat <<EOF
set -e
echo "Checking for $SCAN_HOST from $GR_REMOTE_HOST"
if [ -n "$REMOTE_PORT" ]
then
	if grep -Eq "^\[$REMOTE_HOST\]:$REMOTE_PORT\s" \$HOME/.ssh/known_hosts
	then
		echo "$SCAN_HOST already in known_hosts, exiting"
		exit 0
	fi
else
	if grep -Eq "^$REMOTE_HOST\s" \$HOME/.ssh/known_hosts
	then
		echo "$SCAN_HOST already in known_hosts, exiting"
		exit 0
	fi
fi

echo "Running \"ssh-keyscan $SKARGS $REMOTE_HOST 2>/dev/null >> \$HOME/.ssh/known_hosts\""
touch \$HOME/.ssh/known_hosts # in case it doesn't already exist
ssh-keyscan $SKARGS $REMOTE_HOST 2>/dev/null >> \$HOME/.ssh/known_hosts
EOF
}

if [ -z "$GR_REMOTE_HOST" ]
then
    echo "No remote host given; set GOPHER_REMOTE_HOST or pass '-h <host>'" >&2
    exit 1
fi

SSH_OPTIONS="-o PasswordAuthentication=no"

# Run a remote script
if [ -n "$GR_REMOTE_SCRIPT" ]
then
    if ! [ -e "$GR_REMOTE_SCRIPT" ]
    then
        echo "Remote script \"$GR_REMOTE_SCRIPT\" not found" >&2
        exit 1
    fi
    remote_script | ssh $SSH_OPTIONS $GR_FORWARD_AGENT $GR_REMOTE_USER $GR_REMOTE_HOST
    exit 0
fi

# ssh-keyscan a host remotely
if [ -n "$GR_REMOTE_SCANHOST" ]
then
    remote_scan | ssh $SSH_OPTIONS $GR_FORWARD_AGENT $GR_REMOTE_USER $GR_REMOTE_HOST
    exit 0
fi

REMOTE_COMMAND="$@"
if [ -z "$REMOTE_COMMAND" ]
then
    echo "No command given" >&2
    exit 1
fi

if [ -n "$GR_REMOTE_DIR" ]
then
    REMOTE_COMMAND="cd $GR_REMOTE_DIR; $REMOTE_COMMAND"
fi

ssh $SSH_OPTIONS $GR_FORWARD_AGENT $GR_REMOTE_USER $GR_REMOTE_HOST "$REMOTE_COMMAND"
