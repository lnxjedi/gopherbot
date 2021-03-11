#!/bin/bash

# theia.sh - start theia interface; use ps & kill to terminate

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

FailTask tail-log

if [ ! -e "$HOME/.bashrc" ]
then
    cat > $HOME/.bashrc <<EOF
# File created by jobs/theia.sh; changes will be preserved.
PS1="\[\033[01;32m\]robot@gopherbot-dev\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "
PATH=$HOME/bin:$HOME/.local/bin:$HOME/go/bin:/opt/gopherbot:$PATH
export PATH PS1
EOF
fi

if [ ! -e "$HOME/robot.theia-workspace" ]
then
    cat > $HOME/robot.theia-workspace <<EOF
{
   "folders": [
      {
         "path": "file:///home/robot/custom"
      },
      {
         "path": "file:///home/robot/robot-defaults"
      },
      {
         "path": "file:///home"
      }
   ],
   "settings": {}
}
EOF
fi

cat > $HOME/stop-theia.sh <<"EOF"
kill $PPID
EOF

ln -snf /opt/gopherbot $HOME/robot-defaults || Say "Failed to create symlink $HOME/robot-defaults"

AddTask git-init $GOPHER_CUSTOM_REPOSITORY
AddTask run-theia
