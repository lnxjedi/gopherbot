#!/bin/bash

# publish.sh - copy the install archive to a distribution point

. .publish
VERSTRING=$(grep "var Version" bot/bot.go)
VERSTRING=${VERSTRING#var }
VERSTRING=${VERSTRING// /}
eval $VERSTRING
eval `go env`
BRANCH=$(git rev-parse --abbrev-ref HEAD)

github-release circleci *zip --prerelease --github-repository lnxjedi/gopherbot --tag "circleci"