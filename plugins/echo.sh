#!/bin/bash -e

# echo.sh - trivial shell plugin example for Gopherbot

# Boilerplate
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/pluglib/shellLib.sh

command=$1
shift

case "$command" in
# NOTE: only "configure" should print anything to stdout
	"configure")
		cat <<"EOF"
Channels: [ "botdev" ]
Help:
- Keywords: [ "echo" ]
  Helptext: [ "(bot), echo <simple text> - trivially repeat a phrase" ]
CommandMatches:
- Command: "echo"
  Regex: '(?i:echo ([.;!\d\w-, ]+))'
EOF
		;;
	"echo")
		Reply "$*"
		;;
esac
