#!/bin/bash -e

# auth_setup.sh - Authorizer satisfied when
# $USER_KEY = $SETUP_KEY

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

case $1 in
"configure"|"init")
    exit 0
    ;;
esac

if [ -z "$USER_KEY" -o -z "$SETUP_KEY" ]
then
    exit $PLUGRET_MechanismFail
fi

if [ "$USER_KEY" == "$SETUP_KEY" ]
then
    exit $PLUGRET_Success
else
    exit $PLUGRET_Fail
fi
