#!/bin/bash

docker pull ghcr.io/lnxjedi/gopherbot-dev:latest
docker pull docker.io/buildpack-deps:bullseye
docker build -f containerfile.min -t ghcr.io/lnxjedi/gopherbot:latest .
