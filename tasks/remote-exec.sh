#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

# remote-exec.sh - script for the `remote` task, for running simple commands
# on a remote system. To use the remote task:
# 1) Add the `ssh-init` task to start the robot's ssh agent
# 2) set the following environment vars for the pipeline with
#    `SetParameter <name> <value>`:
# - GOPHER_REMOTE_HOST - required if (-h <host>) not given
# - GOPHER_REMOTE_USER - optional, override with (-l <loginid>); defaults to $USER;
#   if this is always the same, you can also add as a Parameter for the "remote"
#   task
# - GOPHER_REMOTE_DIR - optional remote directory, override with (-d <dir>);
#   defaults to remote $HOME

# Usage: `AddTask remote (-A) (-l <login>) (-h <host>) (-f <file>|-S <scanhost(:port)>|-s|<remote command>)`
#   Executes a command on a remote host. Passing `-A` forwards the robot's ssh agent.
#
#   If `-f <file>` is given, <file> is executed remotely and any further arguments
#   are ignored. NOTE: When "-f <file>" is given, this script prepends "set -e" to the
#   remote script, which causes a non-zero exit on failed commands.
#
#   If `-S <scanhost(:port)` is given, ssh-keyscan is run from the remote
#   host to add a host to known_hosts on the remote system; this should be done ahead
#   of any commands that use ssh remotely. Further arguments are ignored when "-s" is
#   given.
#
#   If '-s' is given, the host keys for the remote host are added to $GOPHER_HOME/known_hosts
#   unless already present.
#
# A standard pipeline using the `remote` task might look like this:
#
    # # Parameters for remote deployment
    # SetParameter GOPHER_REMOTE_HOST my.remote.host
    # SetParameter GOPHER_REMOTE_DIR /var/www/my-application

    # AddTask ssh-init # already done by gopherci in many cases, but good form
    # # Make sure we have the remote system's hostkeys
    # AddTask remote -s
    # # Make sure git pull to git host doesn't fail; "remote -S <host>"
    # # does an ssh-keyscan on the remote host.
    # AddTask remote -S git.my.dom
    # AddTask remote -A git pull
    # AddTask remote ... # more remote commands

unset GR_FORWARD_AGENT GR_REMOTE_HOST GR_REMOTE_SCANHOST GR_REMOTE_SCRIPT GR_REMOTE_USER \
GR_REMOTE_DIR GR_LOCAL_SCAN

[ -n "$GOPHER_REMOTE_USER" ] && GR_REMOTE_USER="-l $GOPHER_REMOTE_USER"
[ -n "$GOPHER_REMOTE_HOST" ] && GR_REMOTE_HOST="$GOPHER_REMOTE_HOST"
[ -n "$GOPHER_REMOTE_DIR" ] && GR_REMOTE_DIR="$GOPHER_REMOTE_DIR"

while getopts ":Al:h:f:S:sd:" OPT
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
        GR_LOCAL_SCAN="true"
        ;;
    S)
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
ssh $SKARGS -o PasswordAuthentication=no -o PubkeyAuthentication=no \
-o StrictHostKeyChecking=no $REMOTE_HOST : 2>&1 || :
EOF
}

if [ -z "$GR_REMOTE_HOST" ]
then
    echo "No remote host given; set GOPHER_REMOTE_HOST or pass '-h <host>'" >&2
    exit 1
fi

if [ "$GR_LOCAL_SCAN" ]
then
    AddTask ssh-scan $GOPHER_REMOTE_HOST
    exit 0
fi

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
