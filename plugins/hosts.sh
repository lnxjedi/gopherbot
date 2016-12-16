#!/bin/bash -e

# hosts.sh - less trivial example shell plugin for gopherbot
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/shellLib.sh

command=$1
shift

configure(){
	cat <<"EOF"
Help:
- Keywords: [ "hosts", "lookup", "dig", "nslookup" ]
  Helptext:
  - "(bot), dig <hostname|ip> ... - lookup a list of hosts and reply with a table of results"
  - "(bot), hosts <hostname|ip> ... - lookup a list of hosts and reply with a table of results"
  - "(bot), hostname - report the $HOSTNAME where the bot is running"
CommandMatches:
- Command: hosts
  Regex: '(?i:hosts?|lookup|dig|nslookup) ([\w-. ]+)'
- Command: hostname
  Regex: '(?i:hostname)'
EOF
}

hosts() {
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
}

case $command in
	"configure")
		configure
		;;
	"hosts")
		hosts $*
		;;
	"hostname")
		Reply "I'm running on $HOSTNAME"
		;;
esac
