#!/bin/sh

default_config() {
cat <<'EOF'
Commands:
- Command: keypair
  Regex: '(?i:(generate|replace) keypair)'
  Keywords: [ "ssh", "keygen", "key", "replace", "keypair" ]
  Usage: "generate keypair"
  Summary: "Creates or replaces the robot's SSH keypair."
  Examples:
  - "(alias) generate keypair"
  - "(bot) replace keypair"
- Command: pubkey
  Regex: '(?i:(show[ -])?pubkey)'
  Keywords: [ "ssh", "pubkey", "public", "key" ]
  Usage: "show pubkey"
  Summary: "Displays the robot's SSH public key."
  Examples:
  - "(alias) pubkey"
  - "(bot) show pubkey"
EOF
}

command=$1
shift

case "$command" in
	configure)
		default_config
		exit 0
		;;
	init)
		exit 0
		;;
esac

set -e

for required in git jq ssh
do
	if ! which "$required" >/dev/null 2>&1
	then
		echo "Required '$required' not found in \$PATH" >&2
		exit 1
	fi
done

if [ -z "$BOT_SSH_PHRASE" ]
then
	say "\$BOT_SSH_PHRASE not set; update Parameters for ssh-init"
	exit 0
fi

ssh_key=${KEYNAME:-robot_key}
ssh_dir=$GOPHER_CONFIGDIR/ssh

case "$command" in
	keypair)
		action=$1
		if [ -e "$ssh_dir/$ssh_key" ]
		then
			if [ "$action" = "replace" ]
			then
				rm -f "$ssh_dir/$ssh_key" "$ssh_dir/$ssh_key.pub"
			else
				say "I've already got an ssh keypair - use 'replace keypair' to replace it"
				exit 0
			fi
		else
			mkdir -p "$ssh_dir"
			chmod 700 "$ssh_dir"
		fi
		bot_name=$(GetBotAttribute name)
		/usr/bin/ssh-keygen -q -b 4096 -N "$BOT_SSH_PHRASE" -C "$bot_name" -f "$ssh_dir/$ssh_key"
		say "Created"
		;;
	pubkey)
		if [ ! -e "$ssh_dir/$ssh_key.pub" ]
		then
			say "I don't seem to have an ssh public key - use 'generate keypair' to create one"
		else
			say -f "$(cat "$ssh_dir/$ssh_key.pub")"
		fi
		;;
	*)
		exit $PLUGRET_NotFound
		;;
esac
