#!/bin/bash -e
# Helper task for adding ssh host keys to known_hosts.
# Usage:
# - Add to ExternalTasks
# - Add it to a pipeline: `AddTask ssh-scan some.host.name`, or `AddTask ssh-scan -p 2022 some.host.name`

SKARGS=""
while [ $# -gt 1 ]
do
	SKARGS="$SKARGS $1"
	shift
done
SKARGS="${SKARGS# }"

echo "Checking for $1 (extra args: $SKARGS)"
if grep -Eq "^$1\s|\[$1\]:" $HOME/.ssh/known_hosts
then
    echo "$1 already in known_hosts, exiting"
	exit 0
fi

echo "Running \"ssh-keyscan $SKARGS $1 2>/dev/null >> $HOME/.ssh/known_hosts\""
touch $HOME/.ssh/known_hosts # in case it doesn't already exist
ssh-keyscan $SKARGS $1 2>/dev/null >> $HOME/.ssh/known_hosts
