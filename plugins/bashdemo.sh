#!/bin/bash -e

# bashdemo - plugin that exercises the bash plugin API, but doesn't do
# anything useful.

# START Boilerplate
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift
# END Boilerplate

configure(){
	cat <<"EOF"
Channels: [ "general", "random" ]
Help:
- Keywords: [ "echo" ]
  Helptext: [ "(bot), echo <simple text> - trivially repeat a phrase" ]
- Keywords: [ "hear" ]
  Helptext: [ "(bot), hear me out - let the robot prove it's really listening" ]
CommandMatchers:
- Regex: '(?i:echo ([.;!\d\w-, ]+))'
  Command: "echo"
- Regex: '(?i:hear me out)'
  Command: "hear"
EOF
}
# TODO: Finish regex/command above

case "$command" in
# NOTE: only "configure" should print anything to stdout
	"configure")
		configure
		;;
	"echo")
		Reply "$*"
		;;
	"hear")
		REPLY=$(PromptForReply "SimpleString" "Well ok then, what do you want to tell me?")
		if [ $? -ne 0 ]
		then
			Reply "Eh, sorry bub, I'm having trouble hearing you - try typing faster?"
		else
			Reply "Ok, I hear you saying \"$REPLY\" - feel better?"
		fi
		;;
esac
