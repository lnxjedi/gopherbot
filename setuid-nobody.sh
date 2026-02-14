#!/bin/bash

# setuid-nobody.sh - trivial convenience script for common permissions on
# gopherbot binary; prevents silly typos.

set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
	echo "Must be run as root"
	exit 1
fi

if ! id nobody >/dev/null 2>&1; then
	echo "User 'nobody' does not exist on this system"
	exit 1
fi

GROUP=""
if command -v getent >/dev/null 2>&1; then
	if getent group nobody >/dev/null 2>&1; then
		GROUP="nobody"
	elif getent group nogroup >/dev/null 2>&1; then
		GROUP="nogroup"
	fi
fi
if [ -z "$GROUP" ]; then
	if id -gn nobody >/dev/null 2>&1; then
		GROUP="$(id -gn nobody)"
	fi
fi
if [ -z "$GROUP" ]; then
	echo "Could not determine group for user 'nobody' (tried: nobody, nogroup, id -gn nobody)"
	exit 1
fi

INSTALLDIR="$(cd -- "$(dirname -- "$0")" && pwd)"
cd "$INSTALLDIR"
if [ ! -f gopherbot ]; then
	echo "Could not find gopherbot binary in $INSTALLDIR"
	exit 1
fi

chown "nobody:$GROUP" gopherbot
chmod 4755 gopherbot
echo "Done. Set owner to nobody:$GROUP and mode to 4755 on $INSTALLDIR/gopherbot"
