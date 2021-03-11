#!/bin/bash

umask 0002

SHELL=/bin/bash
THEIA_DEFAULT_PLUGINS=local-dir:/usr/local/theia/plugins
USE_LOCAL_GIT=true

export SHELL THEIA_DEFAULT_PLUGINS USE_LOCAL_GIT
unset BOT_SSH_PHRASE

cd /usr/local/theia
exec node /usr/local/theia/src-gen/backend/main.js /home/robot --hostname 0.0.0.0
