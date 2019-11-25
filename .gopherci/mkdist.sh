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
Usage: mkdist.sh

Generate distributable .zip file for Linux
EOF
	exit 0
}

if [ "$1" = "-h" -o "$1" = "--help" ]
then
	usage
fi

eval `go env`
COMMIT=$(git rev-parse --short HEAD)

CONTENTS="conf/ doc/ jobs/ lib/ licenses/ plugins/ resources/ robot.skel/ scripts/ \
	tasks/ AUTHORS.txt changelog.txt LICENSE new-robot.sh README.md"
MODULES="goplugins/knock.so goplugins/duo.so goplugins/meme.so goplugins/totp.so \
	connectors/slack.so connectors/rocket.so connectors/terminal.so brains/dynamodb.so"

ADIR="build-archive"
mkdir -p "$ADIR/gopherbot"

BUILDOS="linux"
echo "Building gopherbot for $BUILDOS"
make clean
OUTFILE=../gopherbot-$BUILDOS-$GOARCH.zip
rm -f "$ADIR/gopherbot/gopherbot"
make
cp -a gopherbot "$ADIR/gopherbot/gopherbot"
cp -a $CONTENTS $MODULES "$ADIR/gopherbot"
cd $ADIR
echo "Creating $OUTFILE (from $(pwd))"
zip -r $OUTFILE gopherbot/ --exclude *.swp
tar --exclude *.swp -czf ../gopherbot-$BUILDOS-$GOARCH.tar.gz gopherbot/
cd -

rm -rf "$ADIR"
