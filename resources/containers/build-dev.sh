#!/bin/bash

if [ "$1" -o ! -d tmp ]
then
    rm -rf tmp
    mkdir tmp
    git clone git@github.com:lnxjedi/gopherbot.git tmp/gopherbot
    git clone git@github.com:lnxjedi/gopherbot-doc.git tmp/gopherbot-doc
fi

docker build -f containerfile.dev -t ghcr.io/lnxjedi/gopherbot-dev:latest .
