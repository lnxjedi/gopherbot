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