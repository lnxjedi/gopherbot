#!/bin/bash -e

# echo.sh - less trivial example shell plugin for gopherbot
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/pluglib/shellLib.sh

configure(){
	cat <<"EOF"
Help:
- Keywords: [ "hosts", "lookup", "dig", "nslookup" ]
  Helptext:
  - "(bot), dig <hostname|ip> ... - lookup a list of hosts and reply with a table of results"
  - "(bot), hosts <hostname|ip> ... - lookup a list of hosts and reply with a table of results"
CommandMatches:
- Command: hosts
  Regex: '(?:(?i)hosts?|lookup|dig|nslookup) ([\w-. ]+)'
EOF
}
# Ignore any command but "hosts"
if [ "$1" != "hosts" ]
then
	exit 0
fi
shift

HOSTSARR=($*)
for LOOKUP in "${HOSTSARR[@]}"
do
	ERROR=false
	if echo "$LOOKUP" | grep -qP "[a-zA-Z]+"
	then
		HOSTNAME=$LOOKUP
		IPADDR=$(host $HOSTNAME | grep 'has address') || ERROR=true
		IPADDR=${IPADDR##* }
		[ "$ERROR" = "true" ] && IPADDR="(not found)"
	else
		IPADDR=$LOOKUP
		HOSTNAME=$(host $LOOKUP) || ERROR=true
		HOSTNAME=${HOSTNAME##* }
		HOSTNAME=${HOSTNAME%.}
		[ "$ERROR" = "true" ] && HOSTNAME="(not found)"
	fi
	MESSAGE=$(echo -e "${IPADDR}\t${HOSTNAME}\n$MESSAGE")
done

Say -f "$MESSAGE"
