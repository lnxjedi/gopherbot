#!/bin/bash

if [ ! "$1" ]
then
    echo "Missing required argument <path/to/env>"
    exit 1
fi

docker run --rm --env-file=$1 --name gopherbot-min quay.io/lnxjedi/gopherbot:latest
