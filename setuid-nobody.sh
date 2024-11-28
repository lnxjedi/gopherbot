#!/bin/bash

# setuid-nobody.sh - trivial convenience script for common permissions on
# gopherbot binary; prevents silly typos.

if [ $(id -u) -ne 0 ]
then
	echo "Must be run as root"
	exit 1
fi

INSTALLDIR=$(dirname $0)
cd $INSTALLDIR
chown nobody:nobody gopherbot
chmod u+s gopherbot
echo "Making 'privsep' helper setuid root"
chown root:root privsep
chmod 4755 privsep
echo "Done."
