#!/bin/bash

# ssh-askpass.sh - helper script for ssh-init
# You'll need to store the passphrase with a private message (DM) to the robot:
# - `store task secret ssh-init BOT_SSH_PHRASE=<your-passphrase>`

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -z "$BOT_SSH_PHRASE" ]
then
    Log "Error" "Empty BOT_SSH_PHRASE in ssh-askpass"
    echo ""
    exit 1
fi

cat <<< "$BOT_SSH_PHRASE"
