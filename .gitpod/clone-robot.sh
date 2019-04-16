#!/bin/bash -e

# clone-robot.sh - clone robot.skel or <user>/dev-robot

# lnxjedi/gopherbot should clone robot.skel, and set things
# up for a quick-start

# <user>/gopherbot should clone <user>/dev-robot, which just
# needs dev-robot/.env to get up and running and use for
# development

clear

REMOTE=$(git -C gopherbot/ remote get-url origin)
REMOTE_PREFIX=${REMOTE%/gopherbot.git}
REMOTE_ORG=${REMOTE_PREFIX##*/}

if [ "$REMOTE_ORG" == "lnxjedi" ] # demo
then
    git clone https://github.com/lnxjedi/robot.skel.git
    ln -s ../gopherbot/gopherbot robot.skel/gopherbot
    cat <<EOF

#################################################################

Welcome to the Gopherbot Demo. This script clones an empty robot
configuration repository from lnxjedi/robot.skel, prompts for
required variables, populates robot.skel/.env, and starts a robot
you can play with. (NOTE: you need to click in this tab to enter
input)

EOF
    echo -n "Slack token? (from https://<org>.slack.com/services/new/bot) "
    read TOKEN
    echo -n "Robot name? "
    read BOTNAME
    echo -n "Robot alias? (single character from '\;!&:-', default ';') "
    read ALIAS
    [ -z "$ALIAS" ] && ALIAS=";"
    echo
    cat <<EOF
Your demo robot will log in and be available in the #general channel
for your team, and can also be invited to #random. You can address
your robot with '$BOTNAME, <command>' or '$ALIAS <command>'. To start,
try 'help' for general help, or '$BOTNAME, help' for help on available
commands. Press <enter> to start your demo robot, and <ctrl-c> to exit...
EOF
    read DUMMY
    cat > robot.skel/.env <<EOF
GOPHER_SLACK_TOKEN=$TOKEN
GOPHER_BOTNAME=$BOTNAME
GOPHER_ALIAS=$ALIAS
EOF
    cd robot.skel
    ./gopherbot
else
    git clone $REMOTE_PREFIX/dev-gopherbot.git
cat > start.sh << EOF
#!/bin/bash
if [ ! -d "dev-gopherbot-secrets" ]
then
    git clone $REMOTE_PREFIX/dev-gopherbot-secrets.git
    ln -s ../dev-gopherbot-secrets/environment dev-gopherbot/.env
fi
cd dev-gopherbot
./gopherbot
EOF
    chmod +x start.sh
    ln -s ../gopherbot/gopherbot dev-gopherbot/gopherbot
cat <<EOF

###################################################################################
To start the robot:
$ ./start.sh

Press <enter>
EOF
    read DUMMY
    kill -9 $$
fi