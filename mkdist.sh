#!/bin/bash -e

# mkdist.sh - create a distributable .zip file

usage(){
	cat <<EOF
Usage: mkdist.sh <destfile> (platform)

Generate a distributable .zip file for the given platform. Platform defaults
to "linux", but "macos" can also be specified.
EOF
	exit 0
}

[ $# -eq 0 ] && usage

GOOS="linux"
case "$2" in
	"macos")
		GOOS="darwin"
		;;
esac
export GOOS

echo "Building for $GOOS"
go build

OUTFILE=$1
[[ $1 != *.zip ]] && OUTFILE=$1.zip

echo "Creating $OUTFILE"
zip -r $OUTFILE gopherbot.sh gopherbot LICENSE README.md conf plugins/external util
