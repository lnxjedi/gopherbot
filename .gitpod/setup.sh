#!/bin/bash -e

# setup.sh - set up demo or dev environment

# for main lnxjedi/gopherbot, run new-robot.sh

# for <user>/gopherbot, create start.sh

clear

REMOTE=$(git -C gopherbot/ remote get-url origin)
REMOTE_PREFIX=${REMOTE%/gopherbot.git}
REMOTE_ORG=${REMOTE_PREFIX##*/}

if [ "$REMOTE_ORG" == "lnxjedi" ] # demo
then
    exec ./gopherbot/new-robot.sh demo-robot
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
        exec ./gopherbot/new-robot.sh "\$BOTREPO"
    fi
    ln -s ../gopherbot/gopherbot \$BOTREPO/gopherbot
fi

if [ ! -d "\$CREDREPO" ]
then
    git clone $REMOTE_PREFIX/\$CREDREPO.git || :
    if [ ! -d "\$CREDREPO"]
    then
        echo "Unable to clone $REMOTE_PREFIX/\$CREDREPO.git"
        exit 1
    fi
    ln -s ../\$CREDREPO/environment \$CREDREPO/.env
fi

cd \$BOTREPO
./gopherbot
EOF
    chmod +x start.sh
cat <<EOF

###################################################################################
To start the robot:
$ ./start.sh <botname>

(you can close this tab)
EOF
fi