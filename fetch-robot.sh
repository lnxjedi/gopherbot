#!/bin/bash

# fetch-robot.sh <name>
# Fetch a robot for devel or management

usage(){
    cat <<EOF
Usage: fetch-robot.sh [botname]

Set up the directory structure and fetch repositories for a Gopherbot robot
instance. This only performs 'git clone ...', and assumes proper
credentials are already provided.

Normal operation, without [botname], sources '.env' and expects
GOPHER_CUSTOM_REPOSITORY to be defined there. The [botname] should only be
provided for development environments, where the robot repositories can be
deduced from the repository where the 'fetch-robot.sh' script resides.

Note that this scripts assumes a convention for repository names; for a
robot named 'clu', the following repository names are assumed:
* clu-gopherbot - (required) the custom configuration, including
  'conf/gopherbot.yaml'
* clu-state - (optional) state repository for memories and other cruft
* clu-private - (optional, discouraged) repository containing
  'environment' file with GOPHER_ENCRYPTION_KEY; this repository should
  only be used for development robots, and only if you're exceptionally
  lazy (like me) - please make this repository private
EOF
    exit ${1:-0}
}

SCRIPTPATH=$(readlink -f $0)
SCRIPTDIR=$(dirname $SCRIPTPATH)

devbot(){
    local BOTNAME="$1"
    if ! REMOTE=$(cd $SCRIPTDIR; git remote get-url origin)
    then
        echo "Unable to look up git remote URL"
        exit 1
    fi
    FETCH_PRIVATE="true"
    if ! mkdir -p "$BOTNAME"
    then
        echo "Unable to create directory '$BOTNAME'"
        usage 1
    fi
    GOPHER_CUSTOM_REPOSITORY=${REMOTE/gopherbot/$BOTNAME-gopherbot}
    cd "$BOTNAME"
}

getbot(){
    if ! [ -e ".env" ]
    then
        echo "Missing '.env'"
        usage 1
    fi
    . .env
    if [ -z "$GOPHER_CUSTOM_REPOSITORY" ]
    then
        echo "GOPHER_CUSTOM_REPOSITORY not defined in '.env'"
        usage
    fi
}

case "$1" in
    -h|--help)
        usage
        ;;
    ?*)
        devbot "$1"
        ;;
    *)
        getbot
        ;;
esac

GOPHER_STATE_REPOSITORY=${GOPHER_CUSTOM_REPOSITORY/gopherbot/state}
GOPHER_PRIVATE_REPOSITORY=${GOPHER_CUSTOM_REPOSITORY/gopherbot/private}

echo "Fetching $GOPHER_CUSTOM_REPOSITORY..."
if ! git clone $GOPHER_CUSTOM_REPOSITORY custom
then
    echo "Unable to clone $GOPHER_CUSTOM_REPOSITORY in to 'custom/'"
    usage 1
fi

echo "Fetching $GOPHER_STATE_REPOSITORY..."
if ! git clone $GOPHER_STATE_REPOSITORY state
then
    echo "(not cloned, ignoring)"
fi

echo "Fetching $GOPHER_PRIVATE_REPOSITORY..."
if ! git clone $GOPHER_PRIVATE_REPOSITORY private
then
    echo "(not cloned, ignoring)"
else
    chmod 600 private/environment
fi

ln -snf "$SCRIPTDIR/gopherbot" gopherbot
cat >terminal.sh <<EOF
#!/bin/bash

# terminal.sh - run $BOTNAME in local terminal mode
echo "Setting 'GOPHER_PROTOCOL' to 'terminal' and logging to ./robot.log..."
GOPHER_PROTOCOL=terminal ./gopherbot 2>robot.log
EOF
chmod +x terminal.sh
