#!/bin/bash
# Helper task for adding ssh host keys to known_hosts.
# Usage:
# - Add to ExternalTasks
# - Add it to a pipeline: `AddTask ssh-scan some.host.name`

if grep -q "^$1\s" $HOME/.ssh/known_hosts
then
	exit 0
fi

ssh-keyscan $1 2>/dev/null | grep -v '^#' >> $HOME/.ssh/known_hosts