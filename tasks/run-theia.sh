#!/bin/bash

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

umask 0002

SHELL=/bin/bash
THEIA_DEFAULT_PLUGINS=local-dir:/usr/local/theia/plugins
USE_LOCAL_GIT=true

export SHELL THEIA_DEFAULT_PLUGINS USE_LOCAL_GIT
unset BOT_SSH_PHRASE

cd /usr/local/theia
node /usr/local/theia/src-gen/backend/main.js /home/robot --hostname 0.0.0.0 > /tmp/theia.log 2>/tmp/theia.errorlog &
THEIA_PID=$!

echo "$THEIA_PID" > /tmp/theia.pid
Say "Ok, I started the theia IDE, and you can connect to port 3000"
wait $THEIA_PID || :
