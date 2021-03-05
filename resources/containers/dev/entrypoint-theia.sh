#!/bin/bash
# entrypoint-theia.sh - set the umask, start ssh-agent,
# and start theia

eval `ssh-agent -s`
umask 0002
# This PATH export looks funny, but it basically preserves the PATH
# from the container, which Theia occasionally munges.
cat > $HOME/.bashrc <<EOF
# Created by /usr/local/bin/entrypoint-theia.sh
PS1='[$USER@gopherbot-dev \W]\$ '
PS2='> '
PS4='+ '
export PS1 PS2 PS4
export PATH=$PATH
EOF
# Simpler PATH for Theia to prevent munging; the .bashrc will
# restore the full PATH.
export PATH=/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin
exec node /usr/local/theia/src-gen/backend/main.js "$HOME" --hostname=0.0.0.0
