#!/bin/bash -e

# devbuild.sh - build static gopherbot binary only

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'netgo osusergo static_build' -o gopherbot
