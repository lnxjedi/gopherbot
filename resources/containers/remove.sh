#!/bin/bash -e

# remove.sh - shortcut for docker stop, docker remove

if [ $# -ne 1 ]
then
    echo "Usage: ./remove.sh <name>"
    exit 1
fi

docker stop $1 >/dev/null && docker rm $1 >/dev/null
echo "Removed"
