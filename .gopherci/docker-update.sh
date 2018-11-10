#!/bin/bash

# docker-update.sh - run from root of repository to update the Dockerfiles
# for image rebuilds.

COMMIT=$(git rev-parse --short HEAD)

sed -i "s/ARG commit=.*/ARG commit=$COMMIT/" resources/docker/amazonlinux/Dockerfile
sed -i "s/ARG commit=.*/ARG commit=$COMMIT/" resources/docker/ubuntu/Dockerfile
