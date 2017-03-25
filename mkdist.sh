#!/bin/bash -e

# mkdist.sh - create a distributable .zip file

usage(){
	cat <<EOF
Usage: mkdist.sh

Generate distributable .zip files for the current platform.
EOF
	exit 0
}

VERSTRING=$(grep "var Version" bot/bot.go)
VERSTRING=${VERSTRING#var }
VERSTRING=${VERSTRING// /}
# Set Version
eval $VERSTRING

eval `go env`
echo "Building gopherbot for $GOOS"
go build
OUTFILE=./gopherbot-$Version-$GOOS-$GOARCH.zip
rm -f $OUTFILE

echo "Creating $OUTFILE"
zip -r $OUTFILE gopherbot LICENSE README.md brain/ conf/ doc/ example.gopherbot/ lib/ licenses/ misc/ plugins/ --exclude *.swp doc/.git/\*\* doc/.git/
