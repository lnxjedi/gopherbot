#!/bin/bash

# theia.sh - start theia interface; use ps & kill to terminate

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

FailTask tail-log

mkdir -p "$HOME/.ssh"
chmod 0700 "$HOME/.ssh"
echo "$DEV_PRIVATE_KEY" | base64 -d > $HOME/.ssh/id_code # not really
chmod 0600 "$HOME/.ssh/id_code"

cat > $HOME/.bashrc <<EOF
cat <<WELCOME
Welcome to the Gopherbot Theia IDE.
Use 'stop-theia' to quit.
WELCOME
stop-theia(){
   kill \$(cat /tmp/theia.pid)
}
load-ssh-key(){
   ssh-add \$HOME/.ssh/id_code
}
if ! ssh-add -l &>/dev/null
then
   echo "Use 'load-ssh-key' to load your coding key (\$DEV_KEY_NAME)."
fi
PS1="\[\033[01;32m\]robot@gopherbot-ide\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "
EOF

SetParameter "GOPHERBOT_IDE" "true"

AddTask git-init $GOPHER_CUSTOM_REPOSITORY
AddTask run-theia
