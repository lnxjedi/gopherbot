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

Generate distributable zip/tar.gz files for Linux
EOF
	exit 0
}

if [ "$1" = "-h" -o "$1" = "--help" ]
then
	usage
fi

eval `go env`
COMMIT=$(git rev-parse --short HEAD)

CONTENTS="conf/ doc/ jobs/ lib/ licenses/ plugins/ resources/ robot.skel/ helpers/ \
	tasks/ AUTHORS.txt changelog.txt LICENSE README.md fetch-robot.sh"
MODULES="goplugins/knock.so goplugins/duo.so goplugins/meme.so goplugins/totp.so \
	connectors/slack.so connectors/rocket.so brains/dynamodb.so history/file.so"

ADIR="build-archive"

BUILDOS="linux"

rm -rf "$ADIR/gopherbot"
mkdir -p "$ADIR/gopherbot"
cp -a gopherbot "$ADIR/gopherbot/gopherbot"
cp -a --parents $CONTENTS $MODULES "$ADIR/gopherbot"
cd $ADIR
echo "Creating gopherbot-$BUILDOS-$GOARCH.[zip|tar.gz] (from $(pwd))"
zip -r ../gopherbot-$BUILDOS-$GOARCH.zip gopherbot/ --exclude *.swp *.pyc *__pycache__*
tar --owner=0 --group=0 --exclude *.swp --exclude *.pyc --exclude __pycache__ -czvf ../gopherbot-$BUILDOS-$GOARCH.tar.gz gopherbot/
cd -

rm -rf "$ADIR"
