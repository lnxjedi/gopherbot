#!/bin/bash
# build.sh

eval `go env`
echo "Building for $GOOS"
go build -i -race
