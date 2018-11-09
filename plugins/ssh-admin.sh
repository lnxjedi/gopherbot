#!/bin/bash -e

# ssh-admin.sh - shell plugin for managing the robot's ssh keypair
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

command=$1
shift

configure(){
	cat <<"EOF"
Help:
- Keywords: [ "ssh", "keygen", "key", "replace" ]
  Helptext: [ "(bot), generate|replace keypair" ]
CommandMatchers:
- Command: keypair
  Regex: '(?i:(generate|replace) keypair)'
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
