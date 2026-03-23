#!/bin/sh

set -e

command=$1
shift

case "$command" in
	configure|init)
		exit 0
		;;
esac

FailTask tail-log

case "$command" in
	update)
		say "Ok, I'll trigger the 'updatecfg' job to issue a git pull and reload configuration..."
		AddJob updatecfg
		FailTask say "Job failed!"
		AddTask say "... done"
		;;
	branch)
		branch=$1
		AddJob go-switchbranch "$branch"
		FailTask say "Error switching branches - does '$branch' exist?"
		FailTask tail-log
		AddTask send-message "... switched to branch '$branch'"
		;;
	theia)
		if [ ! -e "/usr/local/theia/src-gen/backend/main.js" ]
		then
			say "Theia installation not found. Wrong container?"
			exit 0
		fi
		say "Ok, I'll start the Theia Gopherbot IDE..."
		AddJob theia
		FailTask say "Starting theia failed! (are you using the gopherbot-theia image?)"
		AddTask say "... Theia finished"
		;;
	*)
		exit $PLUGRET_NotFound
		;;
esac
