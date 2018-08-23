#!/bin/bash

# ansible-playbook.sh - Gopherbot task for running an ansible playbook that uses
# a helper script for supplying a vault passphrase.

# How the sausage is made:
# - The 'ansible-playbook' task is configured with the 'ansible' namespace
# - The bot administrator supplies a vault passphrase with a DM to the robot:
#  - `store task parameter ansible VAULT_PASSWORD=<your-passphrase>` always use
#    the same value
#  - `store repository parameter git.server/my-org/my-repo VAULT_PASSWORD=<your-passphrase'
#    use the value for a given repository only
# - Gopherbot supplies that value in the environment for the task or repository
#   namespace, and the 'vault-password.sh' script just echo's that value

export ANSIBLE_VAULT_PASSWORD_FILE=$GOPHER_INSTALLDIR/scripts/vault-password.sh

exec ansible-playbook "$@"