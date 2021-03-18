#!/bin/bash

# container.sh - simple script for starting a gopherbot container.
# Usage: container.sh (type)
# If type is specified, run the gopherbot-<type> container, otherwise
# just run gopherbot. 

TYPE="gopherbot"
if [ "$1" ]
then
	TYPE="$TYPE-$1"
fi

if [ ! -e ".env" ]
then
	echo "Missing '.env'"
	exit 1
fi

docker pull quay.io/lnxjedi/$TYPE:latest
docker run -it --rm -p=127.0.0.1:3000:3000 --env-file .env --name $TYPE quay.io/lnxjedi/$TYPE:latest
