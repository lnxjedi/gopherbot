#!/bin/sh

set -e

if [ $# -ne 2 ]
then
	Log "Error" "dmnotify called with num args != 2"
	exit 0
fi

user=$1
message=$2
SendUserMessage "$user" "$message"
