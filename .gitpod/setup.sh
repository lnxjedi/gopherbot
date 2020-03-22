#!/bin/bash -e

# setup.sh - run demo bot or give info for devel

clear

REMOTE=$(cd gopherbot; git remote get-url origin)
REMOTE_PREFIX=${REMOTE%/gopherbot.git}
REMOTE_ORG=${REMOTE_PREFIX##*/}

if [ -n "$1" -o "$REMOTE_ORG" == "lnxjedi" ] # demo
then
    cat <<EOF
############################################################################
Welcome to the Gopherbot Demo. This will run Gopherbot
in terminal connector mode, where you can use the 
autosetup plugin to configure a new robot and store it
in a git repository.
############################################################################

EOF
    mkdir demobot
    cd demobot
    exec ../gopherbot/gopherbot
else
cat <<EOF

############################################################################
Fetch your development robot with:
$ ./gopherbot/fetch-robot.sh <botname>

EOF
fi
