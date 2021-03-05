#!/bin/bash
# entrypoint-theia.sh - set the umask, start ssh-agent,
# and start theia

eval `ssh-agent -s`
umask 0002
exec node /usr/local/theia/src-gen/backend/main.js "$HOME" --hostname=0.0.0.0
