#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
Help:
- Keywords: [ "hello", "world" ]
  Helptext: [ "(bot), hello world - the usual first program" ]
CommandMatchers:
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