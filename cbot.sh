#!/bin/bash -e

# botc.sh - Script to simplify running Gopherbot containers

IMAGE_NAME="ghcr.io/lnxjedi/gopherbot-dev"
IMAGE_TAG="latest"

usage() {
    cat <<EOF
Usage: ./botc.sh dev|start|remove

Start development container:
$ ./botc.sh start [-b] [-k <path/to/private_key>] <name> (path/to/bot.env)

Start a gopherbot development container:
 -b - start a bare/bootstrap container without a robot env (no 2nd arg)
 -k <path> - copy ssh private key from given path
 -u - pull the latest version first
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
    echo "Access your dev environment at: http://localhost:7777/?workspace=/home/botdev/gopherbot.code-workspace&tkn=$RANDOM_TOKEN"
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
    if [ "$KEYPATH" ]
    then
        echo "Copying $KEYPATH to $CONTAINERNAME:/home/botdev/.ssh/id_rsa ..."
        docker cp "$KEYPATH" $CONTAINERNAME:/home/botdev/.ssh/id_rsa
        docker exec -it -u root $CONTAINERNAME /bin/bash -c "chown botdev:botdev /home/botdev/.ssh/id_rsa; chmod 0600 /home/botdev/.ssh/id_rsa"
    fi
}

case $COMMAND in
up )
    while getopts ":bk:u" OPT; do
        case $OPT in
        b )
            BARE="true"
            ;;
        k )
            KEYPATH="$OPTARG"
            ;;
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

    if [ $# -eq 0 ]
    then
        echo "Wrong number of arguments given"
        usage
        exit 1
    fi

    IMAGE_SPEC="$IMAGE_NAME:$IMAGE_TAG"

    CONTAINERNAME="$1"

    if STATUS=$(docker inspect -f {{.State.Status}} $CONTAINERNAME 2>/dev/null)
    then
        if [ "$STATUS" == "exited" ]
        then
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

    REQUIRED=2
    [ "$BARE" ] && REQUIRED=1

    if [ $# -ne $REQUIRED ]
    then
        echo "Wrong number of arguments given"
        usage
        exit 1
    fi

    if [ "$PULL" ]
    then
        docker pull $IMAGE_SPEC
    fi

    if [ ! "$BARE" ]
    then
        ENV_FILE_ARG="--env-file $2"
    fi

    RANDOM_TOKEN="$(openssl rand -hex 21)"

    docker run -d -p 127.0.0.1:7777:7777 $ENV_FILE_ARG --name $CONTAINERNAME $IMAGE_SPEC --connection-token $RANDOM_TOKEN

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

