#!/bin/bash -e
# Helper task for adding ssh host keys to known_hosts.
# Usage:
# - Add it to a pipeline: `AddTask ssh-scan some.host.name(:port)`

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

unset REMOTE_PORT

REMOTE_HOST="$1"
SCAN_HOST="$REMOTE_HOST"

if [[ $REMOTE_HOST = *:* ]]
then
	REMOTE_PORT=${REMOTE_HOST##*:}
	REMOTE_HOST=${REMOTE_HOST%%:*}
	SKARGS="-p $REMOTE_PORT"
fi

# Ignore ssh error value; github.com for instance will exit 1
SCAN=$(ssh $SKARGS $SSH_OPTIONS -o PasswordAuthentication=no -o PubkeyAuthentication=no \
-o StrictHostKeyChecking=no $REMOTE_HOST : 2>&1 || :)

if echo "$SCAN" | grep -q "WARNING"
then
	Log "Error" "ssh-scan failed, remote host changed"
	echo "$SCAN" > /dev/stderr
	exit 1
fi
