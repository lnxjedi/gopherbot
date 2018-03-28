#!/bin/bash

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
Help:
- Keywords: [ "echo" ]
  Helptext: [ "(bot), echo <simple text> - trivially repeat a phrase" ]
- Keywords: [ "hear" ]
  Helptext: [ "(bot), hear me out - let the robot prove it's really listening" ]
- Keywords: [ "store", "remember" ]
  Helptext: [ "(bot), store <something> is <something> - store a fact in short-term memory" ]
- Keywords: [ "remember", "recall" ]
  Helptext: [ "(bot), what is <something> - recall a fact / context from short-term memory" ]
CommandMatchers:
- Regex: '(?i:echo ([.;!\d\w-, ]+))'
  Command: "echo"
- Regex: '(?i:hear me out)'
  Command: "hear"
- Regex: '(?i:store ([-\w :\/]+) is ([-\w .,!?:\/]+))'
  Command: "store"
- Regex: '(?i:what is ([-\w :\/]+)\??)'
  Command: "recall"
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
	"store")
		Remember "$1" "$2"
		Say "I'll remember \"$1\" is \"$2\" - but eventually I'll forget!"
		;;
	"recall")
		MEMORY=$(Recall "$1")
		if [ $? -ne 0 ]
		then
			Reply "Gosh, I have no idea - I'm so forgetful!"
		else
			Say "$1 is $MEMORY"
		fi
esac
