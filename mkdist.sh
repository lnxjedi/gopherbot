#!/bin/bash

# mkdist.sh - create a distributable .zip file

usage(){
	cat <<EOF
Usage: mkdist.sh

Generate distributable .zip files for the current platform.
EOF
	exit 0
}

git status | grep -qE "nothing to commit, working directory|tree clean"
if [ $? -ne 0 ]
then
	echo "Your working tree isn't clean, aborting build"
	exit 1
fi

VERSTRING=$(grep "var Version" bot/bot.go)
VERSTRING=${VERSTRING#var }
VERSTRING=${VERSTRING// /}
COMMIT=$(git log -1 | grep commit | cut -f 2 -d' ')
cat >bot/commit.go <<EOF
package bot

func init(){
	commit="$COMMIT"
}
EOF

# Set Version
eval $VERSTRING
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$BRANCH" != "master" ]
then
	Version="$BRANCH"
fi

eval `go env`
for BUILDOS in linux darwin windows
do
	echo "Building gopherbot for $BUILDOS"
	OUTFILE=./gopherbot-$Version-$BUILDOS-$GOARCH.zip
	rm -f $OUTFILE
	if [ "$BUILDOS" = "windows" ]
	then
		GOOS=$BUILDOS go build -o gopherbot.exe 
		echo "Creating $OUTFILE"
		zip -r $OUTFILE gopherbot.exe LICENSE README.md brain/ conf/ doc/ example.gopherbot/ lib/ licenses/ misc/ plugins/ --exclude *.swp doc/.git/\*\* doc/.git/
	else
		GOOS=$BUILDOS go build
		echo "Creating $OUTFILE"
		zip -r $OUTFILE gopherbot LICENSE README.md brain/ conf/ doc/ example.gopherbot/ lib/ licenses/ misc/ plugins/ --exclude *.swp doc/.git/\*\* doc/.git/
	fi

done
rm -f bot/commit.go
