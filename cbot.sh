#!/bin/bash -e

# cbot.sh - Script to simplify running Gopherbot containers

IMAGE_NAME="ghcr.io/lnxjedi/gopherbot"
IMAGE_TAG="latest"

usage() {
    cat <<EOF
Usage: ./cbot.sh <command> (options...) (arguments...)
Use './cbot.sh <command> -h' for help on a given command.
Commands:
- preview: launch the preview for a terminal interface to the default robot
- profile <robotname> <fullname> <email>: generate a robot development profile
- pull: pull the latest docker images
- start <robot.env>: start a development (or production) robot container
- stop <robot.env>: stop a robot container
- remove <robot.env>: stop and remove a robot container
- terminal <robot.env>: shell in to the robot's container
- update: download the latest version of this script from github
- list: generate a list of robot containers
EOF
}

if [ $# -lt 1 ]
then
    usage
    exit 0
fi

COMMAND="$1"
shift

get_access() {
    echo "http://localhost:7777/?workspace=/home/bot/gopherbot.code-workspace&tkn=$RANDOM_TOKEN"
}

show_access() {
    local ENV_TYPE="dev"
    if [ "$1" == "-p" ]
    then
        ENV_TYPE="preview"
        shift
    fi
    GENERATED=$(get_access)
    local ACCESS_URL=${1:-$GENERATED}
    echo "Access your $ENV_TYPE environment at: $ACCESS_URL"
}

check_profile() {
    if [ ! "$GOPHER_PROFILE" ]
    then
        echo "Missing profile argument"
        exit 1
    fi

    if [ ! -e "$GOPHER_PROFILE" ]
    then
        # Check for ${GOPHER_PROFILE}.env file
        if [ -e "${GOPHER_PROFILE}.env" ]
        then
            GOPHER_PROFILE="${GOPHER_PROFILE}.env"
        else
            echo "No profile found for ${GOPHER_PROFILE}"
            exit 1
        fi
    fi
}

read_profile() {
    for CFG_VAR in CONTAINERNAME SSH_KEY_PATH FORWARD_SSH
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
    echo "Copying $SSH_KEY_PATH to $CONTAINERNAME:/home/bot/.ssh/id_ssh ..."
    docker cp "$SSH_KEY_PATH" $CONTAINERNAME:/home/bot/.ssh/id_ssh
    docker exec -it -u root $CONTAINERNAME /bin/bash -c "chown bot:bot /home/bot/.ssh/id_ssh; chmod 0600 /home/bot/.ssh/id_ssh"
}

update_container_uid() {
    EXTERNAL_UID=$(id -u)
    EXTERNAL_GID=$(id -g)
    echo "Updating the container for forwarding the local ssh-agent ..."
    docker exec -u root $CONTAINERNAME /bin/sh -c "sed -i 's/^bot:x:[0-9]*:[0-9]*:/bot:x:$EXTERNAL_UID:$EXTERNAL_GID:/' /etc/passwd"
    docker exec -u root $CONTAINERNAME /bin/sh -c "sed -i 's/^bot:x:[0-9]*:/bot:x:$EXTERNAL_GID:/' /etc/group"
    docker exec -u root $CONTAINERNAME chown -R $EXTERNAL_UID:$EXTERNAL_GID /home/bot /opt
}

case $COMMAND in
profile )
    while getopts ":hfk:" OPT; do
        case $OPT in
        h ) cat <<"EOF"
Generate a profile for working with a gopherbot robot container:
./cbot.sh profile (-k path/to/ssh/private/key) <container-name> "<full name>" <email>
 -k (path) - Load an ssh private key when using this profile
 -f        - Forward the local ssh-agent when using this profile

Example:
$ ./cbot.sh profile -k ~/.ssh/id_rsa bishop "David Parsley" parsley@linuxjedi.org | tee bishop.env
## Lines starting with #| are used by the cbot.sh script
GIT_AUTHOR_NAME="David Parsley"
GIT_AUTHOR_EMAIL=parsley@linuxjedi.org
GIT_COMMITTER_NAME="David Parsley"
GIT_COMMITTER_EMAIL=parsley@linuxjedi.org
#|CONTAINERNAME=bishop
#|SSH_KEY_PATH=/home/david/.ssh/id_rsa
## Items needed for bootstrapping an existing robot
#GOPHER_ENCRYPTION_KEY=<key>
#GOPHER_CUSTOM_REPOSITORY=<git@...> # ssh URL for repo
#GOPHER_PROTOCOL=slack              # if you need to override
#GOPHER_DEPLOY_KEY=<key>
EOF
            exit 0
            ;;
        k )
            SSH_KEY_PATH="$OPTARG"
            ;;
        f )
            FORWARD_SSH="true"
            ;;
        \?)
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
## Items needed for bootstrapping an existing robot
#GOPHER_ENCRYPTION_KEY=<key>
#GOPHER_CUSTOM_REPOSITORY=<git@...> # ssh URL for repo
#GOPHER_PROTOCOL=slack              # if you need to override
#GOPHER_DEPLOY_KEY=<key>
EOF
    if [ "$SSH_KEY_PATH" ]
    then
        echo "#|SSH_KEY_PATH=${SSH_KEY_PATH}"
    fi
    if [ "$FORWARD_SSH" ]
    then
        echo "#|FORWARD_SSH=true"
    fi
    exit 0
    ;;
list | ls )
    while getopts ":h" OPT; do
        case $OPT in
        h ) cat <<"EOF"
List all robot containers:
./cbot.sh list

Example:
$ ./cbot.sh list
CONTAINER ID   STATUS             NAMES        environment         access
0f50a4ce6b2a   Up 37 seconds      bishop-dev   robot/development   http://localhost:7777/?workspace=/home/bot/gopherbot.code-workspace&tkn=XXXXXXX
1c470fd80c31   Up About an hour   clu          robot/production
EOF
            exit 0
            ;;
        \?)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    docker ps -a --filter "label=type=gopherbot/robot" --format "table {{.ID}}\t{{.Status}}\t{{.Names}}\t{{.Label \"environment\"}}\t{{.Label \"access\"}}"
    ;;
remove | rm )
    while getopts ":hp" OPT; do
        case $OPT in
        h ) cat <<"EOF"
Stop and remove a container:
./cbot.sh remove (path/to/profile)
 -p - remove a production robot

Example:
$ ./cbot.sh remove bishop.env
EOF
            exit 0
            ;;
        p )
            PROD="true"
            ;;
        \?)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    shift $((OPTIND -1))
    GOPHER_PROFILE=$1
    check_profile
    eval `read_profile`
    if [ ! "$PROD" ]
    then
        CONTAINERNAME="$CONTAINERNAME-dev"
    fi
    docker stop $CONTAINERNAME >/dev/null && docker rm $CONTAINERNAME >/dev/null
    echo "Removed"
    exit 0
    ;;
stop )
    while getopts ":hp" OPT; do
        case $OPT in
        h ) cat <<"EOF"
Stop a robot container:
./cbot.sh stop (-p) (path/to/profile)
 -p - stop a production robot

Example:
$ ./cbot.sh stop bishop.env
EOF
            exit 0
            ;;
        p )
            PROD="true"
            ;;
        \?)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    shift $((OPTIND -1))
    GOPHER_PROFILE=$1
    check_profile
    eval `read_profile`
    if [ ! "$PROD" ]
    then
        CONTAINERNAME="$CONTAINERNAME-dev"
    fi
    docker stop $CONTAINERNAME >/dev/null
    echo "Stopped"
    exit 0
    ;;
term | terminal )
    while getopts ":hrp" OPT; do
        case $OPT in
        h ) cat <<"EOF"
Shell in to a robot container:
./cbot.sh term (-p) (-r) (path/to/profile)
 -p - look for a production container
 -r - connect as the "root" user

Example:
$ ./cbot.sh term -r bishop.env
EOF
            exit 0
            ;;
        p )
            PROD="true"
            ;;
        r )
            DOCKUSER="-u root"
            ;;
        \?)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    shift $((OPTIND -1))
    GOPHER_PROFILE=$1
    check_profile
    eval `read_profile`
    if [ ! "$PROD" ]
    then
        CONTAINERNAME="$CONTAINERNAME-dev"
    fi
    echo "Starting shell session in the $CONTAINERNAME container ..."
    exec docker exec -it $DOCKUSER $CONTAINERNAME /bin/bash
    ;;
pull )
    while getopts ":h" OPT; do
        case $OPT in
        h ) cat <<"EOF"
Pull the most recent development and production gopherbot containers:
./cbot.sh pull

Example:
$ ./cbot.sh pull
latest: Pulling from lnxjedi/gopherbot-dev
...
EOF
            exit 0
            ;;
        \?)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    docker pull ${IMAGE_NAME}-dev:${IMAGE_TAG}
    docker pull ${IMAGE_NAME}:${IMAGE_TAG}
    exit 0
    ;;
update )
    while getopts ":h" OPT; do
        case $OPT in
        h ) cat <<"EOF"
Download the latest version of the cbot.sh script, replacing the current version:
./cbot.sh update

Example:
$ ./cbot.sh update
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100 12443  100 12443    0     0   276k      0 --:--:-- --:--:-- --:--:--  282k
EOF
            exit 0
            ;;
        \?)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    ## Thanks, OpenAI text-davinci-003!
    # This wasn't added by OpenAI, but it seemed like a good idea to me
    unlink ${BASH_SOURCE[0]}
    # Download the new version of the script directly to the script path
    curl -L "https://raw.githubusercontent.com/lnxjedi/gopherbot/main/cbot.sh" -o ${BASH_SOURCE[0]}

    # Make the script executable
    chmod +x ${BASH_SOURCE[0]}
    exit 0
    ;;
preview )
    CONTAINERNAME='floyd-gopherbot-preview'
    while getopts ":hru" OPT; do
        case $OPT in
        h ) cat <<"EOF"
Preview the Gopherbot IDE and Floyd, the default robot:
./cbot.sh preview (-u) (-r)
 -u - pull the latest container version first
 -r - stop and remove the preview container
(Note: you'll need to connect to the localhost interface, open a terminal,
and run 'gopherbot')
EOF
            exit 0
            ;;
        r )
            docker stop $CONTAINERNAME >/dev/null && docker rm $CONTAINERNAME >/dev/null
            echo "Removed"
            exit 0
            ;;
        u )
            PULL="true"
            ;;
        \?)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done

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
        ACCESS_URL=$(docker inspect --format='{{index .Config.Labels "access"}}' $CONTAINERNAME)
        RANDOM_TOKEN=${ACCESS_URL##*=}
        show_access $ACCESS_URL
        exit 0
    fi

    if [ "$PULL" ]
    then
        docker pull $IMAGE_SPEC
    fi

    echo "Running '$CONTAINERNAME':"

    IMAGE_NAME="$IMAGE_NAME-dev"
    IMAGE_SPEC="$IMAGE_NAME:$IMAGE_TAG"

    RANDOM_TOKEN="$(openssl rand -hex 21)"
    docker run -d \
        -p 127.0.0.1:7777:7777 \
        -l type=gopherbot/robot \
        -l environment=robot/preview \
        -l access=$(get_access) \
        --name $CONTAINERNAME $IMAGE_SPEC \
        --connection-token $RANDOM_TOKEN
    wait_for_container
    if [ ! "$SUCCESS" ]
    then
        echo "Timed out waiting for container to start"
        exit 1
    fi

    show_access -p
    ;;
start )
    while getopts ":hup" OPT; do
        case $OPT in
        h ) cat <<"EOF"
Start a robot container:
./cbot.sh start (-u) (-p) (path/to/profile)
 -u - pull the latest container version first
 -p - start a production robot (minimal image)

Example:
$ ./cbot.sh start bishop.env
Running 'bishop':
Unable to find image 'ghcr.io/lnxjedi/gopherbot-dev:latest' locally
latest: Pulling from lnxjedi/gopherbot-dev
...
Copying /home/david/.ssh/id_rsa to bishop:/home/bot/.ssh/id_ssh ...
Access your dev environment at: http://localhost:7777/?workspace=/home/bot/gopherbot.code-workspace&tkn=XXXXXXX
EOF
            exit 0
            ;;
        u )
            PULL="true"
            ;;
        p )
            PROD="true"
            ;;
        \?)
            [ "$OPT" != "h" ] && echo "Invalid option: $OPTARG"
            usage
            exit 0
            ;;
        esac
    done
    shift $((OPTIND -1))

    GOPHER_PROFILE=$1
    check_profile
    eval `read_profile`

    if [ ! "$PROD" ]
    then
        IMAGE_NAME="$IMAGE_NAME-dev"
        CONTAINERNAME="$CONTAINERNAME-dev"
    fi
    IMAGE_SPEC="$IMAGE_NAME:$IMAGE_TAG"

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
        if [ "$PROD" ]
        then
            echo "... started"
        else
            if [ "$SSH_KEY_PATH" ]
            then
                copy_ssh
            fi
            ACCESS_URL=$(docker inspect --format='{{index .Config.Labels "access"}}' $CONTAINERNAME)
            RANDOM_TOKEN=${ACCESS_URL##*=}
            show_access $ACCESS_URL
        fi
        exit 0
    fi

    if [ "$PULL" ]
    then
        docker pull $IMAGE_SPEC
    fi

    echo "Running '$CONTAINERNAME':"

    if [ "$PROD" ]
    then
        docker run -d \
            --env-file $GOPHER_PROFILE \
            -l type=gopherbot/robot \
            -l environment=robot/production \
            --name $CONTAINERNAME $IMAGE_SPEC
    else
        RANDOM_TOKEN="$(openssl rand -hex 21)"
        if [ "$FORWARD_SSH" ]
        then
            EXTERNAL_UID=$(id -u)
            EXTERNAL_GID=$(id -g)
            SSH_FORWARDING="-u $EXTERNAL_UID:$EXTERNAL_GID -v $(readlink -f $SSH_AUTH_SOCK):/ssh-agent -e SSH_AUTH_SOCK=/ssh-agent"
        else
            CONTAINER_COMMAND=ssh-agent
        fi

        docker run -d $SSH_FORWARDING \
            -p 127.0.0.1:7777:7777 \
            -p 127.0.0.1:8888:8888 \
            --env-file $GOPHER_PROFILE \
            --entrypoint /usr/local/bin/tini \
            -e GOPHER_IDE="$CONTAINERNAME" \
            -l type=gopherbot/robot \
            -l environment=robot/development \
            -l access=$(get_access) \
            --name $CONTAINERNAME $IMAGE_SPEC \
            -- $CONTAINER_COMMAND /bin/sh -c \
            "exec \${OPENVSCODE_SERVER_ROOT}/bin/openvscode-server --host 0.0.0.0 --port 7777 --connection-token=$RANDOM_TOKEN"
    fi
    wait_for_container
    if [ ! "$SUCCESS" ]
    then
        echo "Timed out waiting for container to start"
        exit 1
    fi

    if [ ! "$PROD" ]
    then
        if [ "$SSH_KEY_PATH" ]
        then
            copy_ssh
        elif [ "$FORWARD_SSH" ]
        then
            update_container_uid
        fi
        show_access
    fi
    ;;
* )
    echo "Invalid command: $COMMAND"
    usage
    exit 1
esac
