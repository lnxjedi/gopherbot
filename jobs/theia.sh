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

USERNAME=$(GetSenderAttribute name)
if [ $? -eq $GBRET_Ok ]
then
    cat > $HOME/load-personal-key.sh<<EOF
# Source this in \$HOME: ". load-personal-key.sh"
if [ ! -e "\$HOME/custom/ssh/$USERNAME-key.enc" ]
then
    echo "'\$HOME/custom/ssh/$USERNAME-key.enc' not found."
else
    /opt/gopherbot/gopherbot decrypt -f \$HOME/custom/ssh/$USERNAME-key.enc > \$HOME/coding_key
    chmod 0600 \$HOME/coding_key
    ssh-add -D
    ssh-add \$HOME/coding_key
fi
EOF
fi

AddTask git-init $GOPHER_CUSTOM_REPOSITORY
AddTask run-theia
