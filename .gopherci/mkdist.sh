#!/bin/bash -e
# mkdist.sh - create a distributable gopherbot-*.tar.gz file

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

Generate distributable tar.gz file for Linux
EOF
	exit 0
}

if [ "$1" = "-h" -o "$1" = "--help" ]
then
	usage
fi

eval `go env`

CONTENTS="conf/ jobs/ lib/ licenses/ plugins/ resources/ robot.skel/ helpers/ \
	tasks/ AUTHORS.txt changelog.txt LICENSE README.md setuid-nobody.sh"
MODULES="goplugins/knock.so goplugins/duo.so goplugins/meme.so goplugins/totp.so \
	connectors/slack.so connectors/rocket.so brains/dynamodb.so history/file.so"

ADIR="build-archive"

BUILDOS="linux"

rm -rf "$ADIR/gopherbot"
mkdir -p "$ADIR/gopherbot"
cp -a gopherbot "$ADIR/gopherbot/gopherbot"
cp -a --parents $CONTENTS $MODULES "$ADIR/gopherbot"
chmod -R a+rX $ADIR

cd $ADIR
echo "Creating gopherbot-$BUILDOS-$GOARCH.tar.gz (from $(pwd))"
tar --owner=0 --group=0 --exclude *.swp --exclude *.pyc --exclude __pycache__ -czf ../gopherbot-$BUILDOS-$GOARCH.tar.gz gopherbot/
cd -

rm -rf "$ADIR"
