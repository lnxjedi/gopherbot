#!/bin/sh

set -e

for arg in "$@"
do
	case "$arg" in
		*=*)
			var=${arg%%=*}
			value=${arg#*=}
			export "$var=$value"
			shift
			;;
		*)
			break
			;;
	esac
done

script=$1
shift

case "$script" in
	/*)
		;;
	*/*)
		if [ ! -e "$script" ]
		then
			Log "Warn" "Script not found: $script, ignoring"
			exit 0
		fi
		;;
esac

"$script" "$@"
