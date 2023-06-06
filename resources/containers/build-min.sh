#!/bin/bash

## Normally run in sequence - base, dev, min
# docker pull ghcr.io/lnxjedi/gopherbot-dev:latest
# docker pull docker.io/amazonlinux:2023
docker build -f containerfile.min -t ghcr.io/lnxjedi/gopherbot:latest .
