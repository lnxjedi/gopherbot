#!/bin/bash

# get-version.sh - used by the Makefile to inject the correct
# version in to the Go linker.

if VERSION=$(git describe --exact-match --tags HEAD 2>/dev/null)
then
    echo "$VERSION"
else
    echo "(not set)"
fi
