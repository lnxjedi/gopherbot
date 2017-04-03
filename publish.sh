#!/bin/bash

# publish.sh - copy the install archive to a distribution point

. .publish
VERSTRING=$(grep "var Version" bot/bot.go)
VERSTRING=${VERSTRING#var }
VERSTRING=${VERSTRING// /}
eval $VERSTRING
eval `go env`

SRCFILE=gopherbot-$Version-$GOOS-$GOARCH.zip
echo "Publishing $SRCFILE to $PREFIX/gopherbot/$Version/gopherbot.zip"
aws s3 cp $SRCFILE $PREFIX/gopherbot/$Version/gopherbot.zip
