#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
Channels:
- general
Help:
- Keywords: [ "format", "world" ]
  Helptext: [ "(bot), format world - exercise formatting options" ]
CommandMatchers:
- Regex: '(?i:format world)'
  Command: "format"
EOF
}

case "$COMMAND" in
  "configure")
    configure
    ;;
  "hello")
    # Change default format
    MessageFormat Variable
    PROTO=$(GetBotAttribute protocol)
    Say "Hello, $PROTO World!"
    # Use default format
    Say '_italics_ <one> *bold* `code` @parsley'
    # Raw
    Say -r '_italics_ <one> *bold* `code` @parsley'
    # Variable
    Say -v '_italics_ <one> *bold* `code` @parsley'
    # Fixed
    Say -f '_italics_ <one> *bold* `code` @parsley'
    ;;
esac