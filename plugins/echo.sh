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
- Keywords: [ "repeat" ]
  Helptext: [ "(bot), repeat (me) - prompt for and trivially repeat a phrase" ]
- Keywords: [ "recollect" ]
  Helptext: [ "(bot), recollect - call out to the rubydemo recall command" ]
CommandMatchers:
- Command: "repeat"
  Regex: '(?i:repeat( me)?)'
- Command: "recollect"
  Regex: '(?i:recollect)'
EOF
}

case "$command" in
# NOTE: only "configure" should print anything to stdout
	"configure")
		configure
		;;
	"repeat")
		REPEAT=$(PromptForReply SimpleString "What do you want me to repeat?")
		RETVAL=$?
		if [ $RETVAL -ne $GBRET_Ok ]
		then
			Reply "Sorry, I had a problem getting your reply: $RETVAL"
		else
			Reply "$REPEAT"
		fi
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
