#!/bin/bash -e

# whoami.sh - shell plugin example that retrieves user attributes
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/shellLib.sh

command=$1
shift

configure(){
	cat <<"EOF"
#Channels: [ "botdev" ]
Help:
- Keywords: [ "whoami" ]
  Helptext: [ "(bot), whoami - get the bot to tell you a little about yourself" ]
CommandMatchers:
- Command: "whoami"
  Regex: '(?i:whoami)'
EOF
}

case $command in
	"configure")
		configure
		;;
	"whoami")
		USERFULLNAME=$(GetSenderAttribute fullName)
		USEREMAIL=$(GetSenderAttribute email)
		Reply "You're $USERFULLNAME, $USEREMAIL"
		;;
esac
