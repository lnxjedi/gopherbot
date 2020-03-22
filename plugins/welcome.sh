#!/bin/bash

# welcome.sh - let the default robot greet the user

# START Boilerplate
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift
# END Boilerplate

if [ "$command" == "configure" ]
then
    exit 0
fi

# Note that this plugin is only active when unconfigured and proto == terminal.
if [ "$command" == "init" ]
then
    Pause 1
    NAME=$(GetBotAttribute "name")
    SendChannelMessage "general" "*******"
    SendChannelMessage "general" "Welcome to the *Gopherbot* terminal connector. Since no \
configuration was detected, you're connected to '$NAME', the default robot."
    Pause 2
    ALIAS=$(GetBotAttribute "alias" | tr -d '\0')
    SendChannelMessage "general" "If you've started the robot by mistake, just hit ctrl-D \
to exit and try 'gopherbot --help'; otherwise feel free to play around with the default robot - \
you can start by typing 'help'. If you'd like to start configuring a new robot, \
type: '${ALIAS}setup <protocol>'."
    exit 0
fi
