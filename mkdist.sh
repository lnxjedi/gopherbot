#!/bin/bash -e

# mkdist.sh - create a distributable .zip file

. VERSION

usage(){
	cat <<EOF
Usage: mkdist.sh <destdir>

Generate distributable .zip files for the current platform.
EOF
	exit 0
}

[ $# -eq 0 ] && usage

eval `go env`
echo "Building gopherbot for $GOOS"
go build
OUTFILE=$1/gopherbot-$GOPHERBOT_VERSION-$GOOS-$GOARCH.zip
rm -f $OUTFILE

echo "Creating $OUTFILE"
zip -r $OUTFILE gopherbot LICENSE README.md plugins/ lib/ example.gopherbot/ --exclude *.swp
