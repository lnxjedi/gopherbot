#!/bin/bash

# echo.sh - trivial shell plugin example for Gopherbot

# START Boilerplate
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift
# END Boilerplate

configure(){
	cat <<"EOF"
---
TrustedPlugins:
- rubydemo
- pythondemo
Help:
- Keywords: [ "echo" ]
  Helptext: [ "(bot), echo <simple text> - trivially repeat a phrase" ]
- Keywords: [ "recollect" ]
  Helptext: [ "(bot), recollect - call out to the rubydemo recall command" ]
CommandMatchers:
- Command: "echo"
  Regex: '(?i:echo ([.;!\d\w-, ]+))'
- Command: "recollect"
  Regex: '(?i:recollect)'
EOF
}

case "$command" in
# NOTE: only "configure" should print anything to stdout
	"configure")
		configure
		;;
	"echo")
		Reply "$*"
		;;
	"recollect")
		CallPlugin rubydemo recall
		STATUS=$?
		if [ "$STATUS" -ne "$PLUGRET_Normal" ]
		then
			Say "Dang, there was a problem calling the rubydemo recall command: $STATUS"
		fi
		;;
esac
