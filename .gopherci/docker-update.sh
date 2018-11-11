#!/bin/bash

# docker-update.sh - run from root of repository to update the Dockerfiles
# for image rebuilds.

COMMIT=$(git rev-parse --short HEAD)

sed -i "s/ARG token=.*/ARG token=$COMMIT/" resources/docker/amazonlinux/Dockerfile
sed -i "s/ARG token=.*/ARG token=$COMMIT/" resources/docker/ubuntu/Dockerfile
