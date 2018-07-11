#!/bin/bash

# ssh-init.sh - pipeline job for setting up an ssh-agent for the robot. To
# use:
# - Copy conf/jobs/ssh-init.yaml.sample to ssh-init.yaml (as-is)
# - Add "ssh" and "ssh-agent" to ExternalTask list, with Name, Path and
#   Description (or uncomment)
# - Add job "ssh-init" to ExternalJobs (uncomment from sample) and reload
# - Set the ssh passphrase with a DM to the robot:
#   store parameter ssh-init BOT_SSH_PHRASE=<bot ssh passphrase>
# - Put tasks in a pipeline, e.g.:
#    AddTask ssh-init
#    ... (do stuff)
#    AddTask ssh-agent -k

if [ -z "$BOT_SSH_PHRASE" ]
then
    MESSAGE="BOT_SSH_PHRASE not set, see conf/jobs/ssh-init.yaml"
    Log "Error" "$MESSAGE"
    echo "$MESSAGE" >&2
    exit 1
fi

if [ -z "$HOME" ]
then
    MESSAGE="HOME not set"
    Log "Error" "$MESSAGE"
    echo "$MESSAGE" >&2
    exit 1
fi

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

STMP=$(mktemp -d -p $HOME ssh-agent-XXXXXXX)

cat <<EOF >$STMP/askpass.sh
#!/bin/bash
cat $STMP/sfifo
EOF
chmod +x $STMP/askpass.sh

mkfifo $STMP/sfifo
mkfifo $STMP/afifo
cat <<EOF >$STMP/sfifo &
$BOT_SSH_PHRASE
EOF

ssh-agent >$STMP/afifo &
eval `cat $STMP/afifo`

export SSH_ASKPASS=$STMP/askpass.sh
export DISPLAY=""
SSH_KEY=${KEYNAME:-id_rsa}
ssh-add $HOME/.ssh/$SSH_KEY < /dev/null

# Make agent available to other tasks in the pipeline
SetParameter SSH_AUTH_SOCK $SSH_AUTH_SOCK
SetParameter SSH_AGENT_PID $SSH_AGENT_PID

rm -rf $STMP

exit 0