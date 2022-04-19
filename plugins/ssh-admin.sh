#!/bin/bash -e

# ssh-admin.sh - shell plugin for managing the robot's ssh keypair
# see tasks/ssh-init.sh

command=$1
shift

configure(){
	cat <<"EOF"
Help:
- Keywords: [ "ssh", "keygen", "key", "replace", "keypair" ]
  Helptext: [ "(bot), generate|replace keypair - create an ssh keypair for the robot" ]
- Keywords: [ "ssh", "pubkey", "public", "key" ]
  Helptext: [ "(bot), (show) pubkey - show the robot's public key" ]
CommandMatchers:
- Command: keypair
  Regex: '(?i:(generate|replace) keypair)'
- Command: pubkey
  Regex: '(?i:(show[ -])?pubkey)'
EOF
}

if [ "$command" = "configure" ]
then
	configure
	exit 0
elif [ "$command" = "init" ]
then
	exit 0
fi

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}" >&2
    # The script should usually exit on error
    exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }
source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

for REQUIRED in git jq ssh
do
    if ! which $REQUIRED >/dev/null 2>&1
    then
        echo "Required '$REQUIRED' not found in \$PATH"
        exit 1
    fi
done

if [ -z "$BOT_SSH_PHRASE" ]
then
	Say "\$BOT_SSH_PHRASE not set; update Parameters for ssh-init"
	exit 0
fi

SSH_KEY=${KEYNAME:-robot_key}

SSH_DIR=$GOPHER_CONFIGDIR/ssh

case $command in
	"keypair")
		ACTION=$1
		if [ -e $SSH_DIR/$SSH_KEY ]
		then
			if [ "$ACTION" = "replace" ]
			then
				rm -f $SSH_DIR/$SSH_KEY $SSH_DIR/$SSH_KEY.pub
			else
				Say "I've already got an ssh keypair - use 'replace keypair' to replace it"
				exit 0
			fi
		else
			mkdir -p $SSH_DIR
			chmod 700 $SSH_DIR
		fi
		BOT=$(GetBotAttribute name)
		/usr/bin/ssh-keygen -q -b 4096 -N "$BOT_SSH_PHRASE" -C "$BOT" -f $SSH_DIR/$SSH_KEY
		Say "Created"
		;;
	"pubkey")
		if [ ! -e $SSH_DIR/$SSH_KEY.pub ]
		then
			Say "I don't seem to have an ssh public key - use 'generate keypair' to create one"
		else
			Say -f "$(cat $SSH_DIR/$SSH_KEY.pub)"
		fi
		;;
esac
