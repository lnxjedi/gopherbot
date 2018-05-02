#!/bin/bash -e

# githubci.sh - a Bash plugin for triggering a build pipeline
# when Slack commit notifications are received from the Slack
# Github app.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
Users:
- github # override this with your own local config if needed
# Channels: [ "your", "channels" ] # override
MessageMatchers:
- Command: "build"
  Regex: 'new commit.*github.com\/(.*)\/tree\/(.*)\|'
EOF
}

case "$COMMAND" in
	"configure")
		configure
		;;
  "build")
    REPO="$1"
    BRANCH="$2"
    Say "Hey! I see there's a new commit to '$REPO' in the '$BRANCH' branch. Someday I'm going to do something about that!"
    ;;
esac