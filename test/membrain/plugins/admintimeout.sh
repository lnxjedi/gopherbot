#!/bin/bash

[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source "$GOPHER_INSTALLDIR/lib/gopherbot_v1.sh"

command=$1
shift

configure() {
	cat <<"EOF"
---
Commands:
- Command: "slow"
  Regex: '(?i:admin slow)'
- Command: "fail"
  Regex: '(?i:admin fail)'
AllowedHiddenCommands:
- slow
- fail
EOF
}

case "$command" in
	"configure")
		configure
		;;
	"slow")
		echo "slow stdout before sleep"
		echo "slow stderr before sleep" >&2
		sleep 10
		Say "slow done"
		;;
	"fail")
		echo "Traceback (most recent call last):" >&2
		echo "  File \"admintimeout.py\", line 1, in <module>" >&2
		echo "RuntimeError: boom" >&2
		exit 1
		;;
esac
