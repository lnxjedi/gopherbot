#!/bin/bash
# build.sh

eval `go env`
echo "Building race-detecting gopherbot for $GOOS"
go install -race
