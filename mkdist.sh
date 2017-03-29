#!/bin/bash

# mkdist.sh - create a distributable .zip file

usage(){
	cat <<EOF
Usage: mkdist.sh

Generate distributable .zip files for the current platform.
EOF
	exit 0
}

git status | grep -q "nothing to commit, working tree clean"
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

eval `go env`
echo "Building gopherbot for $GOOS"
go build
OUTFILE=./gopherbot-$Version-$GOOS-$GOARCH.zip
rm -f $OUTFILE

echo "Creating $OUTFILE"
zip -r $OUTFILE gopherbot LICENSE README.md brain/ conf/ doc/ example.gopherbot/ lib/ licenses/ misc/ plugins/ --exclude *.swp doc/.git/\*\* doc/.git/
