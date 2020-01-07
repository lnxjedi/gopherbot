#!/bin/bash -e

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
Help:
- Keywords: [ "repeat" ]
  Helptext: [ "(bot), repeat (me) - prompt for and trivially repeat a phrase" ]
- Keywords: [ "echo" ]
  Helptext: [ "(bot), echo <something> - tell the bot to say <something>" ]
CommandMatchers:
- Command: "repeat"
  Regex: '(?i:repeat( me)?)'
- Command: "echo"
  Regex: '(?i:echo (.*))'
EOF
}

case "$command" in
# NOTE: only "configure" should print anything to stdout
	"configure")
		configure
		;;
	"echo")
		Pause 1 # because the robot knows how to "type"
		Say "$1"
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
esac
