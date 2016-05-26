#!/bin/bash -e

# mkdist.sh - create a distributable .zip file

GOPHERBOT_VERSION=0.8

usage(){
	cat <<EOF
Usage: mkdist.sh <destdir>

Generate distributable .zip files for the supported platforms,
currently linux and macos (darwin).
EOF
	exit 0
}

[ $# -eq 0 ] && usage

for PLATFORM in darwin linux
do
	./build.sh $PLATFORM
	OUTFILE=$1/gopherbot-$GOPHERBOT_VERSION-$PLATFORM.zip
	rm -f $OUTFILE

	echo "Creating $OUTFILE"
	zip -r $OUTFILE gopherbot LICENSE README.md conf/ plugins/ util/ gopherbot.template/ --exclude *.swp
done
