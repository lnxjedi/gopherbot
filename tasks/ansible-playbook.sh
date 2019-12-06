#!/bin/bash

# ansible-playbook.sh - Gopherbot task for running an ansible playbook that uses
# a helper script for supplying a vault passphrase.

# Note that "VAULT_PASSWORD" needs to be stored as an encrypted parameter for either
# ansible-playbook or for a given repository in repositories.yaml.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ -n "$VAULT_PASSWORD" ]
then
    export ANSIBLE_VAULT_PASSWORD_FILE=$GOPHER_INSTALLDIR/helpers/vault-password.sh
else
    Log "Warn" "No VAULT_PASSWORD secret found for job ${GOPHER_JOB_NAME:-(none)} / extended namespace ${GOPHER_NAMESPACE_EXTENDED:-(none)}"
fi

exec ansible-playbook "$@"
