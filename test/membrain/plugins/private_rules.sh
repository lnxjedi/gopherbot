#!/bin/bash

[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source "$GOPHER_INSTALLDIR/lib/gopherbot_v1.sh"

command=$1

case "$command" in
	"free"|"secret"|"locked"|"open")
		Reply "private-rules:${command}"
		;;
	"fieldtest")
		Reply "Fields: $2 then $3"
		;;
esac

exit 0
