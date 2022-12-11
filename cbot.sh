#!/bin/bash -e

# cbot.sh - Script to simplify running Gopherbot containers

IMAGE_NAME="ghcr.io/lnxjedi/gopherbot-dev"
IMAGE_TAG="latest"

usage() {
    cat <<EOF
Usage: ./botc.sh profile|dev|start|stop|remove (options...) (arguments...)

----
Generate a profile for development:
./cbot.sh profile (-k path/to/ssh/private/key) <container-name> "<full name>" <email>
 -k (path) - Load an ssh private key when using this profile

Example:
$ ./cbot.sh profile -k ~/.ssh/id_rsa bishop "David Parsley" parsley@linuxjedi.org | tee ~/bishop.env
## Lines starting with #| are used by the cbot.sh script
GIT_AUTHOR_NAME="David Parsley"
GIT_AUTHOR_EMAIL=parsley@linuxjedi.org
GIT_COMMITTER_NAME="David Parsley"
GIT_COMMITTER_EMAIL=parsley@linuxjedi.org
#|CONTAINERNAME=bishop
#|SSH_KEY_PATH=/home/david/.ssh/id_rsa

----
Start a gopherbot development container:
./cbot.sh dev (-u) (path/to/profile)
 -u - pull the latest container version first

Example:
$ ./cbot.sh dev ~/bishop.env
Running 'bishop':
Unable to find image 'ghcr.io/lnxjedi/gopherbot-dev:latest' locally
latest: Pulling from lnxjedi/gopherbot-dev
...
Copying /home/david/.ssh/id_rsa to bishop:/home/bot/.ssh/id_ssh ...
Access your dev environment at: http://localhost:7777/?workspace=/home/bot/gopherbot.code-workspace&tkn=XXXXXXX

----
Stop a gopherbot container:
./cbot.sh stop (path/to/profile)

Example:
$ ./cbot.sh stop ~/bishop.env

----
Stop and remove a container:
./cbot.sh remove (path/to/profile)

Example:
$ ./cbot.sh remove ~/bishop.env
EOF
}

if [ $# -lt 1 ]
then
    usage
    exit 0
fi

COMMAND="$1"
shift

show_access() {
    echo "Access your dev environment at: http://localhost:7777/?workspace=/home/bot/gopherbot.code-workspace&tkn=$RANDOM_TOKEN"
}

check_profile() {
    if [ ! "$GOPHER_PROFILE" ]
    then
        echo "Missing profile argument" 
        exit 1
    fi

    if [ ! -e "$GOPHER_PROFILE" ]
    then
        echo "Profile file not found: $GOPHER_PROFILE"
        exit 1
    fi
}

read_profile() {
    for CFG_VAR in CONTAINERNAME SSH_KEY_PATH
    do
        RAW=$(grep "^#|$CFG_VAR" $GOPHER_PROFILE)
        echo "${CFG_VAR}=${RAW#*=}"
    done
}

wait_for_container() {
    # Give it a minute to start running
    for TRY in {1..60}
    do
        if [ "`docker inspect -f {{.State.Running}} $CONTAINERNAME`"=="true" ]
        then
            SUCCESS="true"
            break
        fi
        sleep 1
    done
}

copy_ssh() {
    if [ "$SSH_KEY_PATH" ]
    then
        echo "Copying $SSH_KEY_PATH to $CONTAINERNAME:/home/bot/.ssh/id_ssh ..."
        docker cp "$SSH_KEY_PATH" $CONTAINERNAME:/home/bot/.ssh/id_ssh
        docker exec -it -u root $CONTAINERNAME /bin/bash -c "chown bot:bot /home/bot/.ssh/id_ssh; chmod 0600 /home/bot/.ssh/id_ssh"
    fi
}

case $COMMAND in
profile )
    while getopts ":k:" OPT; do
        case $OPT in
        k )
            SSH_KEY_PATH="$OPTARG"
            ;;
        \? | h)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    shift $((OPTIND -1))
    if [ $# -ne 3 ]
    then
        echo "Wrong number of arguments"
        usage
        exit 1
    fi
    CONTAINERNAME="$1"
    GIT_USER="$2"
    GIT_EMAIL="$3"
    cat <<EOF
## Lines starting with #| are used by the cbot.sh script
GIT_AUTHOR_NAME="${GIT_USER}"
GIT_AUTHOR_EMAIL=${GIT_EMAIL}
GIT_COMMITTER_NAME="${GIT_USER}"
GIT_COMMITTER_EMAIL=${GIT_EMAIL}
#|CONTAINERNAME=${CONTAINERNAME}
EOF
    if [ "$SSH_KEY_PATH" ]
    then
        echo "#|SSH_KEY_PATH=${SSH_KEY_PATH}"
    fi
    exit 0
    ;;
remove | rm )
    GOPHER_PROFILE=$1
    check_profile
    eval `read_profile`
    docker stop $CONTAINERNAME >/dev/null && docker rm $CONTAINERNAME >/dev/null
    echo "Removed"
    exit 0
    ;;
stop )
    GOPHER_PROFILE=$1
    check_profile
    eval `read_profile`
    docker stop $CONTAINERNAME >/dev/null
    echo "Stopped"
    exit 0
    ;;
dev )
    while getopts ":u" OPT; do
        case $OPT in
        u )
            PULL="true"
            ;;
        \? | h)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    shift $((OPTIND -1))

    IMAGE_SPEC="$IMAGE_NAME:$IMAGE_TAG"

    GOPHER_PROFILE=$1
    check_profile
    eval `read_profile`

    if STATUS=$(docker inspect -f {{.State.Status}} $CONTAINERNAME 2>/dev/null)
    then
        echo "(found existing container '$CONTAINERNAME', re-using)"
        if [ "$STATUS" == "exited" ]
        then
            echo "Starting '$CONTAINERNAME':"
            docker start $CONTAINERNAME
            wait_for_container
            if [ ! "$SUCCESS" ]
            then
                echo "Timed out waiting for container to start"
                exit 1
            fi
        fi
        copy_ssh
        TOK_LINE=$(docker logs $CONTAINERNAME 2>/dev/null | grep "^Web UI" | tail -1)
        RANDOM_TOKEN=${TOK_LINE##*=}
        show_access
        exit 0
    fi

    if [ "$PULL" ]
    then
        docker pull $IMAGE_SPEC
    fi

    RANDOM_TOKEN="$(openssl rand -hex 21)"

    echo "Running '$CONTAINERNAME':"
    docker run -d \
        -p 127.0.0.1:7777:7777 \
        -p 127.0.0.1:8888:8888 \
        --env-file $GOPHER_PROFILE \
        --name $CONTAINERNAME $IMAGE_SPEC \
        --connection-token $RANDOM_TOKEN

    wait_for_container
    if [ ! "$SUCCESS" ]
    then
        echo "Timed out waiting for container to start"
        exit 1
    fi

    copy_ssh
    show_access
    ;;
* )
    echo "Invalid command: $COMMAND"
    usage
    exit 1
esac

