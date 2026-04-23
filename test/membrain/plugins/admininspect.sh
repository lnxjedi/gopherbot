#!/bin/bash

[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source "$GOPHER_INSTALLDIR/lib/gopherbot_v1.sh"

command=$1
shift

configure() {
	cat <<"EOF"
---
Commands:
- Command: "inspect"
  Regex: '(?i:admin inspect)'
AllowedHiddenCommands:
- inspect
EOF
}

case "$command" in
	"configure")
		configure
		;;
	"inspect")
		echo "inspect stdout ready"
		echo "inspect stderr ready" >&2
		sleep 1
		Say "inspect done"
		;;
esac
