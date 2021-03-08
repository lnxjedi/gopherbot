#!/bin/bash

docker run --rm -p=127.0.0.1:3000:3000 --name gopherbot-dev quay.io/lnxjedi/gopherbot-dev:latest
