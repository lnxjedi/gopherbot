#!/bin/bash

# theia.sh - start theia interface; use ps & kill to terminate

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

FailTask tail-log

if [ ! -e "$HOME/robot.theia-workspace" ]
then
    cat > $HOME/robot.theia-workspace <<EOF
{
   "folders": [
      {
         "path": "file:///var/lib/robot/custom"
      },
      {
         "path": "file:///var/lib/robot/robot-defaults"
      }
   ],
   "settings": {}
}
EOF
fi

cat > $HOME/stop-theia.sh <<"EOF"
# Source this in $HOME: ". stop-theia.sh"
kill $PPID
EOF

mkdir -p "$HOME/.ssh"
chmod 0700 "$HOME/.ssh"
echo "$DEV_PRIVATE_KEY" | base64 -d > $HOME/.ssh/id_code # not really
chmod 0600 "$HOME/.ssh/id_code"

cat > $HOME/.bashrc <<EOF
cat <<WELCOME
Welcome to the Gopherbot IDE. Use 'ssh-add .ssh/id_code' to
load your ssh key, or '. stop-theia.sh' to stop the IDE.
WELCOME
EOF

SetParameter "GOPHERBOT_IDE" "true"

AddTask git-init $GOPHER_CUSTOM_REPOSITORY
AddTask run-theia
