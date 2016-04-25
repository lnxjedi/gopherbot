#!/bin/bash
# build.sh - build the robot executable for a given platform

GOOS=${1:-linux}
export GOOS

echo "Building for $GOOS"
go build -o robot
