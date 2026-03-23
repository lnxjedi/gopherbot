#!/bin/sh

set -e

if [ -n "$GOPHER_USER" ]
then
	Reply "$*"
else
	say "$*"
fi
