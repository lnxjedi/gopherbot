#!/bin/bash

# test.sh - the code here is more of use for the test suite than as a good
# source of copy-n-paste code. ymmv. Note the lack of helptext.

# START Boilerplate
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift
# END Boilerplate

configure(){
	cat <<"EOF"
---
CommandMatchers:
- Command: "waitask"
  Regex: '(?i:waitask)'
- Command: "asknow"
  Regex: '(?i:asknow)'
EOF
}

case "$command" in
# NOTE: only "configure" should print anything to stdout
	"configure")
		configure
		;;
	"waitask")
		(sleep 3; Say "ok - answer puppies") &
		sleep 2
		# The 'bot will have to wait to hear back about kittens
		REPLY=$(PromptForReply YesNo "Do you like kittens?")
		# Make sure this isn't said at the same time as the next question
		Say "I like kittens too!"
		;;
	"asknow")
		REPLY=$(PromptForReply YesNo "Do you like puppies?")
		sleep 1
		Say "I like puppies too!"		
esac
