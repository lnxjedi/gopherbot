#!/bin/bash

# ssh-askpass.sh - helper script for ssh-init

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -z "$BOT_SSH_PHRASE" ]
then
    Log "Error" "Empty BOT_SSH_PHRASE in ssh-askpass"
    echo "Missing BOT_SSH_PHRASE"
    exit 1
fi

cat <<< "$BOT_SSH_PHRASE"
