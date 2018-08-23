#!/bin/bash

# vault-password.sh - Ansible helper script for supplying a vault passphrase.
# You'll need to store the passphrase with a DM to the robot:
# - `store task parameter ansible VAULT_PASSWORD=<your-passphrase>` always use
#   the same value
# - `store repository parameter git.server/my-org/my-repo VAULT_PASSWORD=<your-passphrase'
#   use the value for a given repository only
# Used in conjunction with tasks/ansible-playbook.sh

echo "$VAULT_PASSWORD"