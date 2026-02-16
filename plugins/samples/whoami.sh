#!/bin/bash -e

# whoami.sh - shell plugin example that retrieves user attributes
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift

configure(){
	cat <<"EOF"
Commands:
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
