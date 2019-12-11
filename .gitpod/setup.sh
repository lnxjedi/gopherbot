#!/bin/bash -e

# setup.sh - run demo bot or give info for devel

clear

REMOTE=$(cd gopherbot; git remote get-url origin)
REMOTE_PREFIX=${REMOTE%/gopherbot.git}
REMOTE_ORG=${REMOTE_PREFIX##*/}

if [ -n "$1" -o "$REMOTE_ORG" == "lnxjedi" ] # demo
then
    cat <<EOF
Welcome to the Gopherbot Demo. This will run Gopherbot
in terminal connector mode, and eventually allow you to
configure a new robot. This is a work in progress.

Type `help` for general help, or `;quit` to exit.

EOF
    exec ./gopherbot/gopherbot -l /dev/stderr 2> robot.log
else
cat <<EOF

############################################################################
Fetch your development robot with:
$ ./gopherbot/fetch-robot.sh <botname>

EOF
fi
