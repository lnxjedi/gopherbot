#!/bin/bash

# vault-password.sh - Ansible helper script for supplying a vault passphrase.
# You'll need to store the passphrase with a DM to the robot:
# `store parameter ansible:<your-repository> VAULT_PASSWORD=<your-passphrase>
# Used in conjunction with tasks/ansible-playbook.sh

echo "$VAULT_PASSWORD"