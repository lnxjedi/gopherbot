#!/bin/bash

if [ ! "$1" ]
then
    echo "Missing required argument <path/to/env>"
    exit 1
fi

docker run -it --rm -p=127.0.0.1:3000:3000 --env-file=$1 --name gopherbot-theia quay.io/lnxjedi/gopherbot-theia:latest
