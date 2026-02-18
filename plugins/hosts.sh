#!/bin/bash -e

# hosts.sh - less trivial example shell plugin for gopherbot
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift

configure(){
	cat <<"EOF"
Commands:
- Command: hosts
  Regex: '(?i:hosts?|lookup|dig|nslookup) ([\w-. ]+)'
  Keywords: [ "hosts", "lookup", "dig", "nslookup" ]
  Usage: "hosts <hostname|ip> ..."
  Summary: "Looks up one or more hostnames/IPs and returns IP-to-name results."
  Examples:
  - "(alias) hosts github.com 8.8.8.8"
  - "(bot) dig api.example.com"
  Helptext:
  - "(bot), dig <hostname|ip> ... - lookup a list of hosts and reply with a table of results"
  - "(bot), hosts <hostname|ip> ... - lookup a list of hosts and reply with a table of results"
- Command: hostname
  Regex: '(?i:hostname)'
  Keywords: [ "hosts", "hostname" ]
  Usage: "hostname"
  Summary: "Reports the host name where the robot process is running."
  Examples:
  - "(alias) hostname"
  Helptext:
  - "(bot), hostname - report the $HOSTNAME where the bot is running"
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
