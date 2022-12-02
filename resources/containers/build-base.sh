#!/bin/bash

docker pull docker.io/buildpack-deps:bullseye-curl
docker build -f containerfile.base -t ghcr.io/lnxjedi/gopherbot-base:latest .
