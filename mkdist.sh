#!/bin/bash -e

# mkdist.sh - create a distributable .zip file

GOPHERBOT_VERSION=0.8

usage(){
	cat <<EOF
Usage: mkdist.sh <destdir>

Generate distributable .zip files for the current platform.
EOF
	exit 0
}

[ $# -eq 0 ] && usage

eval `go env`
./build.sh
OUTFILE=$1/gopherbot-$GOPHERBOT_VERSION-$GOOS.zip
rm -f $OUTFILE

echo "Creating $OUTFILE"
zip -r $OUTFILE gopherbot LICENSE README.md conf/ plugins/ util/ example.gopherbot/ --exclude *.swp
