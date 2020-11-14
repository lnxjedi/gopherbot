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

CONTENTS="conf/ doc/ jobs/ lib/ licenses/ plugins/ resources/ robot.skel/ helpers/ \
	tasks/ AUTHORS.txt changelog.txt LICENSE README.md fetch-robot.sh"
MODULES="goplugins/knock.so goplugins/duo.so goplugins/meme.so goplugins/totp.so \
	connectors/slack.so connectors/rocket.so brains/dynamodb.so history/file.so"

ADIR="build-archive"

for BUILDOS in ${1:-linux darwin}
do
	echo "Building gopherbot for $BUILDOS"
	make clean
	OUTFILE=../gopherbot-$BUILDOS-$GOARCH.zip
	rm -rf "$ADIR/gopherbot"
	mkdir -p "$ADIR/gopherbot"
	if [ "$BUILDOS" = "linux" ]
	then
		make
	else
		GOOS=$BUILDOS make static
	fi
	cp -a gopherbot "$ADIR/gopherbot/gopherbot"
	if [ "$BUILDOS" = "linux" ]
	then
		cp -a --parents $CONTENTS $MODULES "$ADIR/gopherbot"
	else
		cp -a --parents $CONTENTS "$ADIR/gopherbot"
	fi
	cd $ADIR
	echo "Creating $OUTFILE (from $(pwd))"
	zip -r $OUTFILE gopherbot/ --exclude *.swp *.pyc *__pycache__*
	tar --owner=0 --group=0 --exclude *.swp --exclude *.pyc --exclude __pycache__ -czvf ../gopherbot-$BUILDOS-$GOARCH.tar.gz gopherbot/
	cd -
done

rm -rf "$ADIR"
