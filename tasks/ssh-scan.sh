#!/bin/bash -e
# Helper task for adding ssh host keys to known_hosts.
# Usage:
# - Add it to a pipeline: `AddTask ssh-scan some.host.name(:port)`

unset REMOTE_PORT

REMOTE_HOST="$1"
SCAN_HOST="$REMOTE_HOST"

if [[ $REMOTE_HOST = *:* ]]
then
	REMOTE_PORT=${REMOTE_HOST##*:}
	REMOTE_HOST=${REMOTE_HOST%%:*}
	SKARGS="-p $REMOTE_PORT"
fi

echo "Checking for $SCAN_HOST"
if [ -n "$REMOTE_PORT" ]
then
	if grep -Eq "^\[$REMOTE_HOST\]:$REMOTE_PORT\s" $HOME/.ssh/known_hosts
	then
		echo "$SCAN_HOST already in known_hosts, exiting"
		exit 0
	fi
else
	if grep -Eq "^$REMOTE_HOST\s" $HOME/.ssh/known_hosts
	then
		echo "$SCAN_HOST already in known_hosts, exiting"
		exit 0
	fi
fi

echo "Running \"ssh-keyscan $SKARGS $REMOTE_HOST 2>/dev/null >> $HOME/.ssh/known_hosts\""
touch $HOME/.ssh/known_hosts # in case it doesn't already exist
ssh-keyscan $SKARGS $REMOTE_HOST 2>/dev/null >> $HOME/.ssh/known_hosts
