#!/bin/bash

# generate-secret.sh - generate the secrets for your robot from a .env

usage(){
    echo "Usage:"
    echo "$ ./generate-secret.sh <name> <env file>"
    exit 1
}

if [ $# -ne 2 ]
then
    usage
fi

SECNAME="$1"
EFILE="$2"

if [ ! -f $EFILE ]
then
    usage
fi

kubectl create secret generic $SECNAME --from-env-file $EFILE --dry-run=client -o yaml
