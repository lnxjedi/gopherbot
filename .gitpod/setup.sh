#!/bin/bash -e

# setup.sh - set up demo or dev environment for gitpod

# for main lnxjedi/gopherbot, run new-robot.sh

# for <user>/gopherbot, create start.sh

clear

REMOTE=$(git -C gopherbot/ remote get-url origin)
REMOTE_PREFIX=${REMOTE%/gopherbot.git}
REMOTE_ORG=${REMOTE_PREFIX##*/}

if [ -n "$1" -o "$REMOTE_ORG" == "lnxjedi" ] # demo
then
    cat <<EOF
Welcome to the Gopherbot Demo. If you have a Slack token,
you can connect the demo robot to your Slack team using the
'slack' protocol. Otherwise, you can use the 'term' protocol
to try Gopherbot with the terminal connector.

EOF
    echo -n "Protocol? (one of: slack, term) "
    read PROTOCOL
    exec ./gopherbot/new-robot.sh demo-robot $PROTOCOL
else
cat > start.sh <<EOF
#!/bin/bash -e

BOTNAME=\$1

if [ -z "\$BOTNAME" ]
then
    echo "Usage: ./start.sh <botname>"
    exit 1
fi

BOTREPO="\$BOTNAME-gopherbot"
CREDREPO="\$BOTNAME-credentials"

if [ ! -d "\$BOTREPO" ]
then
    git clone $REMOTE_PREFIX/\$BOTREPO.git || :
    if [ ! -d "\$BOTREPO" ]
    then
        cat <<HEOF
Repository \$BOTREPO not found. Try:
$ ./gopherbot/new-robot.sh \$BOTREPO
HEOF
        exit 1
    fi
    ln -s ../gopherbot/gopherbot \$BOTREPO/gopherbot
fi

if [ ! -d "\$CREDREPO" ]
then
    git clone $REMOTE_PREFIX/\$CREDREPO.git || :
    if [ ! -d "\$CREDREPO" ]
    then
        echo "Unable to clone $REMOTE_PREFIX/\$CREDREPO.git"
        exit 1
    fi
    ln -s ../\$CREDREPO/environment \$BOTREPO/.env
fi

cd \$BOTREPO
./gopherbot
EOF
    chmod +x start.sh
cat <<EOF

###################################################################################
To start the robot:
$ ./start.sh <botname>

start.sh will:
- clone $REMOTE_PREFIX/<botname>-gopherbot.git
  -> robot configuration repository
- clone $REMOTE_PREFIX/<botname>-credentials.git
  -> 'environment' file with credentials and secrets
- create symlinks in <botname>-gopherbot/ for the gopherbot binary and .env
- start the robot

The development robot can be restarted with 'cd <botname>-gopherbot; ./gopherbot'

(you can close this tab)
EOF
fi