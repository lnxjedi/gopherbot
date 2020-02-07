#!/bin/bash

# vault-password.sh - Ansible helper script for supplying a vault passphrase.
# You'll need to store the encrypted passphrase as a VAULT_PASSWORD parameter
# in repositories.yaml
#
# Used in conjunction with tasks/ansible-playbook.sh

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -z "$VAULT_PASSWORD" ]
then
    Log "Error" "Empty VAULT_PASSWORD in vault-password.sh, needs encrypted parameter in repositories.yaml"
    echo ""
    exit 1
fi

cat <<< "$VAULT_PASSWORD"
