#!/bin/bash

# heroku-app-vars.sh - convenience script for configuring Gopherbot env vars.

usage(){
    echo "Usage: heroku-app-vars.sh <app-name> (path-to-env)"
    echo "  If '(path-to-env)' not specified, uses '.env'"
    exit 1
}

if [ "$#" -lt 1 ]
then
    echo "App name not given"
    usage
fi

export HEROKU_APP=$1

ENV_FILE=".env"
if [ "$2" ]
then
    ENV_FILE="$2"
fi

if [ ! -e "$ENV_FILE" ]
then
    echo "Environment file not found: $ENV_FILE"
    usage
fi

echo "Using '$ENV_FILE' for env vars ..."
source "$ENV_FILE"

for VAR in GOPHER_ENCRYPTION_KEY GOPHER_CUSTOM_REPOSITORY GOPHER_PROTOCOL GOPHER_DEPLOY_KEY
do
    heroku config:set $VAR="${!VAR}"
done
