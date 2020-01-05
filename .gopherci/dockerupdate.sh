#!/bin/bash

# dockerupdate.sh - update gopherbot-docker after successful build and publish

COMMIT=$(git rev-parse --short HEAD)

rm -rf gopherbot-docker/*

echo "$COMMIT" > gopherbot-docker/gopherbot-commit
cp -a resources/docker/* gopherbot-docker/

cd gopherbot-docker

git add -A
git commit -m "Update on successful build of commit $COMMIT"
git remote set-url origin git@github.com:lnxjedi/gopherbot-docker.git
git push
