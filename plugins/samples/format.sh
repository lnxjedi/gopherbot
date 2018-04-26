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
- Regex: '(?i:format fixed)'
  Command: "fixed"
- Regex: '(?i:format variable)'
  Command: "variable"
- Regex: '(?i:format raw)'
  Command: "raw"
EOF
}

case "$COMMAND" in
  "configure")
    configure
    ;;
  "format")
    # Change default format
    MessageFormat Variable
    PROTO=$(GetBotAttribute protocol)
    Say "Hello, $PROTO World!"
    # Use default format
    Say '_Italics_ <One> *Bold* `Code` @parsley'
    # Raw
    Say -r '_Italics_ <One> *Bold* `Code` @parsley'
    # Variable
    Say -v '_Italics_ <One> *Bold* `Code` @parsley'
    # Fixed
    Say -f '_Italics_ <One> *Bold* `Code` @parsley'
    ;;
  "fixed")
    Say -f '_Italics_ <One> *Bold* `Code` @parsley'
    ;;
  "variable")
    Say -v '_Italics_ <One> *Bold* `Code` @parsley'
    ;;
  "raw")
    Say -r '_Italics_ <One> *Bold* `Code` @parsley'
    ;;
esac