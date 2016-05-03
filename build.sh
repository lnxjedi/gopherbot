#!/bin/bash
# build.sh - build the robot executable for a given platform

GOOS=${1:-linux}
export GOOS
if [ "$GOOS" = "linux" ]
then
	BUILDARGS="-i -race"
fi

echo "Building for $GOOS"
go build -o robot $BUILDARG
