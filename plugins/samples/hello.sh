#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
Commands:
- Regex: '(?i:hello world)'
  Command: "hello"
AmbientMatchCommand: true
MessageMatchers:
- Regex: 'hello robot'
  Command: "hello"
EOF
}

case "$COMMAND" in
    "configure")
        configure
        ;;
    "hello")
        Say "Hello, World!"
        ;;
esac