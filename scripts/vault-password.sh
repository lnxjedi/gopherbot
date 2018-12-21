#!/bin/bash

# vault-password.sh - Ansible helper script for supplying a vault passphrase.
# You'll need to store the passphrase with a DM to the robot:
# - `store repository secret git.server/my-org/my-repo VAULT_PASSWORD=<your-passphrase'
#
# Used in conjunction with tasks/ansible-playbook.sh

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

VAULT_PASSWORD=$(GetSecret VAULT_PASSWORD)
if [ -z "$VAULT_PASSWORD" ]
then
    Log "Error" "Empty VAULT_PASSWORD in vault-password.sh"
    echo ""
    exit 1
fi

echo "$VAULT_PASSWORD"
