#!/bin/sh

default_config() {
cat <<'YAML'
---
Commands:
  - Regex: (?i:gsh local demo)(?: (.*))?
    Command: demo
  - Regex: (?i:gsh local prompt)
    Command: prompt
  - Regex: (?i:gsh local memory)
    Command: memory
  - Regex: (?i:gsh local config)
    Command: config
YAML
}

demo() {
	sender=$(GetSenderAttribute fullName)
	color=$(GetParameter CAT_COLOR)
	say "GSH demo for ${sender}; cat color=${color}."
	Reply "First capture arg was ${2:-<none>}"
	SendChannelMessage "$GOPHER_CHANNEL" "channel message from GSH"
	return "$PLUGRET_Normal"
}

prompt_demo() {
	name=$(PromptForReply SimpleString "Name of cat?") || return "$PLUGRET_Fail"
	say "GSH heard cat name: ${name}"
	return "$PLUGRET_Normal"
}

memory_demo() {
	Remember last_cat Garfield
	Remember team_cat Pixel true
	local_cat=$(Recall last_cat)
	team_cat=$(Recall team_cat true)
	say "GSH memory local=${local_cat} shared=${team_cat}"
	return "$PLUGRET_Normal"
}

config_demo() {
	cfg=$(GetTaskConfig) || return "$PLUGRET_Fail"
	opening=$(printf '%s' "$cfg" | jq -r '.Openings[0]')
	say "GSH config opening: ${opening}"
	SetParameter GSH_LOCAL_DEMO ok
	AddTask next-task from-gsh
	return "$PLUGRET_Normal"
}

case "${1:-}" in
	_configure)
		default_config
		;;
	_init)
		exit "$PLUGRET_Normal"
		;;
	demo)
		demo "$@"
		;;
	prompt)
		prompt_demo
		;;
	memory)
		memory_demo
		;;
	config)
		config_demo
		;;
	*)
		Log error "unknown GSH local command: ${1:-}"
		exit "$PLUGRET_Fail"
		;;
esac
