#!/bin/bash -e

# setup.sh - set up demo or dev environment for gitpod

# for main lnxjedi/gopherbot, run new-robot.sh

# for <user>/gopherbot, create start.sh

clear

REMOTE=$(git remote get-url origin)
REMOTE_PREFIX=${REMOTE%/gopherbot.git}
REMOTE_ORG=${REMOTE_PREFIX##*/}

if [ -n "$1" -o "$REMOTE_ORG" == "lnxjedi" ] # demo
then
    cat <<EOF
Welcome to the Gopherbot Demo. This will run Gopherbot
in terminal connector mode, and eventually allow you to
configure a new robot. This is a work in progress.

Type `help` for general help, or `;quit` to exit.

EOF
    exec ./gopherbot -l /dev/stderr 2> robot.log
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
CREDREPO="\$BOTNAME-private"

if [ ! -d "../\$BOTNAME" ]
then
    mkdir ../\$BOTNAME
    git clone $REMOTE_PREFIX$BOTREPO.git ../\$BOTNAME/custom || :
    git clone $REMOTE_PREFIX$CREDREPO.git ../\$BOTNAME/private || :
    if [ ! -d "../\$BOTREPO" ]
    then
        cat <<HEOF
Repository \$BOTREPO not found.
HEOF
        exit 1
    fi
    ln -s ../gopherbot/gopherbot ../\$BOTREPO/gopherbot
fi

cd ../\$BOTREPO
./gopherbot
EOF
    chmod +x start.sh
cat <<EOF

###################################################################################
To start the robot:
$ ./start.sh <botname>

start.sh will:
- clone $REMOTE_PREFIX/<botname>-gopherbot.git to ../<botname>/custom
  -> robot configuration repository
- clone $REMOTE_PREFIX/<botname>-private.git (PRIVATE git repository) to
  ../<botname>/private
  -> 'environment' file with GOPHER_ENCRYPTION_KEY
- create a symlink in ../<botname>/ for the gopherbot binary
- start the robot

The development robot can be restarted with 'cd ../<botname>; ./gopherbot'

(you can close this tab)
EOF
fi