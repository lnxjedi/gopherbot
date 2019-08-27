#!/bin/bash -e

# auth_setup.sh - Authorizer satisfied when
# $USER_KEY = $SETUP_KEY, or GOPHER_USER
# is the GOPHER_USER we remember.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

case $1 in
"configure"|"init")
    exit 0
    ;;
esac

AUTH_USER=$(Recall "AUTH_USER")
if [ "$GOPHER_USER" == "$AUTH_USER" ]
then
    exit $PLUGRET_Success
fi

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
