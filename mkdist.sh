#!/bin/bash -e

# mkdist.sh - create a distributable .zip file

. VERSION

usage(){
	cat <<EOF
Usage: mkdist.sh

Generate distributable .zip files for the current platform.
EOF
	exit 0
}

eval `go env`
echo "Building gopherbot for $GOOS"
go build
OUTFILE=./gopherbot-$GOPHERBOT_VERSION-$GOOS-$GOARCH.zip
rm -f $OUTFILE

echo "Creating $OUTFILE"
zip -r $OUTFILE gopherbot LICENSE README.md plugins/ lib/ example.gopherbot/ --exclude *.swp
