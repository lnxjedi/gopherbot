#!/bin/bash -e
# mkdist.sh - create a distributable .zip file

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}"
    exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

usage(){
	cat <<EOF
Usage: mkdist.sh (linux|darwin|windows)

Generate distributable .zip files for the given platform, or all platforms if
no argument given.
EOF
	exit 0
}

if [ "$1" = "-h" -o "$1" = "--help" ]
then
	usage
fi

eval `go env`
PLATFORMS=${1:-linux darwin}
COMMIT=$(git rev-parse --short HEAD)

CONTENTS="conf/ doc/ jobs/ lib/ licenses/ plugins/ resources/ robot.skel/ scripts/ tasks/ AUTHORS.txt changelog.txt LICENSE new-robot.sh README.md"

ADIR="build-archive"
mkdir -p "$ADIR/gopherbot"
cp -a $CONTENTS "$ADIR/gopherbot"

for BUILDOS in $PLATFORMS
do
	echo "Building gopherbot for $BUILDOS"
	OUTFILE=../gopherbot-$BUILDOS-$GOARCH.zip
	rm -f $OUTFILE
	rm -f "$ADIR/gopherbot/gopherbot"
	if [ "$BUILDOS" = "linux" ]
	then
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -ldflags "-X main.Commit=$COMMIT" -a -tags 'netgo osusergo static_build' -o gopherbot
		cp -a gopherbot "$ADIR/gopherbot/gopherbot"
		cd $ADIR
		echo "Creating $OUTFILE (from $(pwd))"
		zip -r $OUTFILE gopherbot/ --exclude *.swp
		tar --exclude *.swp -czf ../gopherbot-$BUILDOS-$GOARCH.tar.gz gopherbot/
		cd -
	else
		GOOS=$BUILDOS go build -mod vendor -ldflags "-X main.Commit=$COMMIT"
		cp -a gopherbot "$ADIR/gopherbot/gopherbot"
		cd $ADIR
		echo "Creating $OUTFILE (from $(pwd))"
		zip -r $OUTFILE $ARCHIVE --exclude *.swp gopherbot/
		cd -
	fi
done

rm -rf "$ADIR"
