#!/bin/sh

default_config() {
cat <<'YAML'
---
Commands:
  - Regex: (?i:gsh default hello)
    Command: hello
YAML
}

hello() {
	bot_name=$(GetBotAttribute fullName)
	sender=$(GetSenderAttribute fullName)
	say "GSH says hello from ${bot_name} to ${sender} in #${GOPHER_CHANNEL}."
	Reply "This script only needs the installed default fixture."
	return "$PLUGRET_Normal"
}

case "${1:-}" in
	_configure)
		default_config
		;;
	_init)
		exit "$PLUGRET_Normal"
		;;
	hello)
		hello
		;;
	*)
		Log error "unknown GSH default command: ${1:-}"
		exit "$PLUGRET_Fail"
		;;
esac
