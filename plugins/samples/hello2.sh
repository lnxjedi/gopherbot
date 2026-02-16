#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
Channels:
- random
AllowDirect: false
Commands:
- Regex: '(?i:hello robot)'
  Command: "hello"
EOF
}

case "$COMMAND" in
    "configure")
        configure
        ;;
    "hello")
        Say "I'm here"
        ;;
esac