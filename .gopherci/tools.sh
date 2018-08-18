#!/bin/bash

# tools.sh - install tools needed for gopherci pipeline

# go get is broken without this for now, blocker for go1.12
export GO111MODULE=off
go get github.com/lnxjedi/github-release
#go get github.com/mattn/goveralls
