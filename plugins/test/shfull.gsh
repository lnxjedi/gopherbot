#!/bin/sh

default_config() {
cat <<'EOF'
---
Commands:
- Regex: (?i:say everything)
  Command: sendmsg
- Regex: (?i:sh-config)
  Command: configtest
- Regex: (?i:sh-subscribe)
  Command: subscribe
- Regex: (?i:sh-prompts)
  Command: prompts
- Regex: (?i:sh-memory-seed)
  Command: memoryseed
- Regex: (?i:sh-memory-check)
  Command: memorycheck
- Regex: (?i:sh-memory-thread-check)
  Command: memorythreadcheck
- Regex: (?i:sh-memory-delete)
  Command: memorydelete
- Regex: (?i:sh-memory-thread-delete)
  Command: memorythreaddelete
- Regex: (?i:sh-identity)
  Command: identity
- Regex: (?i:sh-parameter-addtask)
  Command: parameteraddtask
- Regex: (?i:sh-utils)
  Command: utilities
- Regex: (?i:sh-admin-check)
  Command: admincheck
- Regex: (?i:sh-elevate-check)
  Command: elevatecheck
- Regex: (?i:sh-pipeline-ok)
  Command: pipelineok
- Regex: (?i:sh-pipeline-fail)
  Command: pipelinefail
- Regex: (?i:sh-spawn-job)
  Command: spawnjob
- Regex: (?i:pc-add-cmd)
  Command: pipeaddcmd
- Regex: (?i:pc-final-cmd)
  Command: pipefinalcmd
- Regex: (?i:pc-fail-cmd)
  Command: pipefailcmd
AllowedHiddenCommands:
- sendmsg
Config:
  Openings:
  - "Not completely random 1"
  - "Not completely random 2"
EOF
}

show_memory() {
	if [ -z "$1" ]
	then
		printf '<empty>'
	else
		printf '%s' "$1"
	fi
}

sendmsg() {
	say "Regular Say"
	SayThread "SayThread, yeah"
	Reply "Regular Reply"
	ReplyThread "Reply in thread, yo"
	SendChannelMessage "$GOPHER_CHANNEL" "Sending to the channel: $GOPHER_CHANNEL"
	SendUserMessage "$GOPHER_USER" "Sending this message to user: $GOPHER_USER"
	SendUserChannelMessage "$GOPHER_USER" "$GOPHER_CHANNEL" "Sending to user '$GOPHER_USER' in channel: $GOPHER_CHANNEL"
	SendChannelThreadMessage "$GOPHER_CHANNEL" "$GOPHER_THREAD_ID" "Sending to channel '$GOPHER_CHANNEL' in thread: $GOPHER_THREAD_ID"
	SendUserChannelThreadMessage "$GOPHER_USER" "$GOPHER_CHANNEL" "$GOPHER_THREAD_ID" "Sending to user '$GOPHER_USER' in channel '$GOPHER_CHANNEL' in thread: $GOPHER_THREAD_ID"
	return $PLUGRET_Normal
}

configtest() {
	cfg=$(GetTaskConfig) || return $PLUGRET_Fail
	case "$cfg" in
		*"Not completely random 1"*)
			say "Not completely random 1"
			;;
		*"Not completely random 2"*)
			say "Not completely random 2"
			;;
		*)
			say "No config available"
			return $PLUGRET_Fail
			;;
	esac
	return $PLUGRET_Normal
}

subscribe() {
	sub=$(Subscribe)
	unsub=$(Unsubscribe)
	say "SUBSCRIBE FLOW: ${sub}/${unsub}"
	return $PLUGRET_Normal
}

prompts() {
	p1=$(PromptForReply "SimpleString" "Codename check: pick a mission codename.")
	ret=$?
	[ $ret -eq $GBRET_Ok ] || { say "PROMPT FLOW FAILED 1:${ret}"; return $PLUGRET_Fail; }

	p2=$(PromptThreadForReply "SimpleString" "Thread check: pick a favorite snack for launch.")
	ret=$?
	[ $ret -eq $GBRET_Ok ] || { say "PROMPT FLOW FAILED 2:${ret}"; return $PLUGRET_Fail; }

	p3=$(PromptUserForReply "SimpleString" "$GOPHER_USER" "DM check: name a secret moon base.")
	ret=$?
	[ $ret -eq $GBRET_Ok ] || { say "PROMPT FLOW FAILED 3:${ret}"; return $PLUGRET_Fail; }

	p4=$(PromptUserChannelForReply "SimpleString" "$GOPHER_USER" "$GOPHER_CHANNEL" "Channel check: describe launch weather in two words.")
	ret=$?
	[ $ret -eq $GBRET_Ok ] || { say "PROMPT FLOW FAILED 4:${ret}"; return $PLUGRET_Fail; }

	p5=$(PromptUserChannelThreadForReply "SimpleString" "$GOPHER_USER" "$GOPHER_CHANNEL" "$GOPHER_THREAD_ID" "Thread rally: choose a backup call sign.")
	ret=$?
	[ $ret -eq $GBRET_Ok ] || { say "PROMPT FLOW FAILED 5:${ret}"; return $PLUGRET_Fail; }

	say "PROMPT FLOW OK: ${p1} | ${p2} | ${p3} | ${p4} | ${p5}"
	return $PLUGRET_Normal
}

memoryseed() {
	Remember "launch_snack" "saffron noodles"
	Remember "launch_snack" "solar soup" true
	RememberContext "pad" "orbital-7"
	RememberThread "thread_note" "delta thread"
	RememberContextThread "mission" "aurora mission"
	say "MEMORY SEED: done"
	return $PLUGRET_Normal
}

memorycheck() {
	local_mem=$(Recall "launch_snack")
	shared_mem=$(Recall "launch_snack" true)
	ctx_mem=$(Recall "context:pad")
	thread_mem=$(Recall "thread_note")
	thread_ctx=$(Recall "context:mission")
	say "MEMORY CHECK: local=$(show_memory "$local_mem") shared=$(show_memory "$shared_mem") ctx=$(show_memory "$ctx_mem") thread=$(show_memory "$thread_mem") threadctx=$(show_memory "$thread_ctx")"
	return $PLUGRET_Normal
}

memorythreadcheck() {
	local_mem=$(Recall "launch_snack")
	shared_mem=$(Recall "launch_snack" true)
	ctx_mem=$(Recall "context:pad")
	thread_mem=$(Recall "thread_note")
	thread_ctx=$(Recall "context:mission")
	say "MEMORY THREAD CHECK: local=$(show_memory "$local_mem") shared=$(show_memory "$shared_mem") ctx=$(show_memory "$ctx_mem") thread=$(show_memory "$thread_mem") threadctx=$(show_memory "$thread_ctx")"
	return $PLUGRET_Normal
}

memorydelete() {
	DeleteMemory "launch_snack"
	DeleteMemory "launch_snack" true
	DeleteMemory "context:pad"
	say "MEMORY DELETE: done"
	return $PLUGRET_Normal
}

memorythreaddelete() {
	DeleteMemory "thread_note"
	DeleteMemory "context:mission"
	say "MEMORY THREAD DELETE: done"
	return $PLUGRET_Normal
}

identity() {
	bot_name=$(GetBotAttribute name); bot_ret=$?
	sender_name=$(GetSenderAttribute firstname); sender_ret=$?
	bob_name=$(GetUserAttribute bob firstname); bob_ret=$?
	if SetParameter PIPELINE_SENTINEL nebula-42
	then
		set_result=true
	else
		set_result=false
	fi
	param_value=$(GetParameter PIPELINE_SENTINEL)
	say "IDENTITY CHECK: bot=${bot_name}/$(ret_name $bot_ret) sender=${sender_name}/$(ret_name $sender_ret) bob=${bob_name}/$(ret_name $bob_ret) set=${set_result} param=$(show_memory "$param_value")"
	return $PLUGRET_Normal
}

ret_name() {
	case "$1" in
		0) printf 'Ok' ;;
		3) printf 'AttributeNotFound' ;;
		*) printf '%s' "$1" ;;
	esac
}

parameteraddtask() {
	if SetParameter PIPELINE_SENTINEL nebula-42
	then
		say "SETPARAM ADDTASK: queued"
		AddTask param-show
		return $PLUGRET_Normal
	fi
	say "SETPARAM ADDTASK: failed"
	return $PLUGRET_Fail
}

utilities() {
	tmpdir=$(mktemp -d "$GOPHER_WORKSPACE/shfull.XXXXXX") || return $PLUGRET_Fail
	mkdir -p "$tmpdir/a" || return $PLUGRET_Fail
	printf 'beta\nalpha\nbeta\n' > "$tmpdir/a/input.txt"
	cp "$tmpdir/a/input.txt" "$tmpdir/a/copy.txt" || return $PLUGRET_Fail
	mv "$tmpdir/a/copy.txt" "$tmpdir/a/moved.txt" || return $PLUGRET_Fail
	touch "$tmpdir/a/marker.txt" || return $PLUGRET_Fail

	head_line=$(head -n 1 "$tmpdir/a/moved.txt")
	tail_line=$(tail -n 1 "$tmpdir/a/moved.txt")
	line_info=$(wc -l "$tmpdir/a/moved.txt")
	set -- $line_info
	line_count=$1
	uniq_lines=$(cat "$tmpdir/a/moved.txt" | sort | uniq | tr '\n' ',')
	printf 'ship' | base64 > "$tmpdir/a/encoded.txt"
	decoded=$(base64 -d "$tmpdir/a/encoded.txt")
	printf '{"phase":"go"}\n' > "$tmpdir/a/data.json"
	jq_phase=$(jq -r '.phase' "$tmpdir/a/data.json") || return $PLUGRET_Fail
	gzip "$tmpdir/a/moved.txt" || return $PLUGRET_Fail
	gunzip "$tmpdir/a/moved.txt.gz" || return $PLUGRET_Fail
	recovered=$(cat "$tmpdir/a/moved.txt")
	base_name=$(basename "$tmpdir/a/moved.txt")
	dir_name=$(basename "$(dirname "$tmpdir/a/moved.txt")")
	find_info=$(find "$tmpdir" -name '*.txt' | wc -l)
	set -- $find_info
	find_count=$1
	which_info=$(which say)
	case "$which_info" in
		*"gsh builtin"*) which_ok=yes ;;
		*) which_ok=no ;;
	esac
	cd "$tmpdir/a" || return $PLUGRET_Fail
	pwd_name=$(basename "$(pwd)")
	seq_info=$(seq 2 4 | tr '\n' ',')
	yes_info=$(yes hi | head -n 2 | tr '\n' ',')
	env_ok=no
	if env | grep -q 'GOPHER_WORKSPACE='
	then
		env_ok=yes
	fi
	say -f "UTILS OK: head=${head_line} tail=${tail_line} lines=${line_count} uniq=${uniq_lines} decode=${decoded} jq=${jq_phase} recovered=${recovered} base=${base_name} dir=${dir_name} find=${find_count} which=${which_ok} pwd=${pwd_name} seq=${seq_info} yes=${yes_info} env=${env_ok}"
	cd / || return $PLUGRET_Fail
	rm -r "$tmpdir"
	return $PLUGRET_Normal
}

admincheck() {
	admin=$(CheckAdmin)
	say "ADMIN CHECK: ${admin}"
	return $PLUGRET_Normal
}

elevatecheck() {
	if Elevate true
	then
		result=true
	else
		result=false
	fi
	say "ELEVATE CHECK: ${result}"
	return $PLUGRET_Normal
}

pipelineok() {
	AddTask pipeline-note add-task
	AddJob pipe-job job-step
	AddCommand shfull pc-add-cmd
	FinalTask pipeline-note final-task
	FinalCommand shfull pc-final-cmd
	say "PIPELINE OK: queued"
	return $PLUGRET_Normal
}

pipelinefail() {
	FailTask pipeline-note fail-task
	FailCommand shfull pc-fail-cmd
	say "PIPELINE FAIL: armed"
	return $PLUGRET_Fail
}

spawnjob() {
	SpawnJob pipe-spawn-job spawn-step
	return $PLUGRET_Normal
}

pipeaddcmd() {
	say "PIPE ADD COMMAND: ran"
	return $PLUGRET_Normal
}

pipefinalcmd() {
	say "PIPE FINAL COMMAND: ran"
	return $PLUGRET_Normal
}

pipefailcmd() {
	say "PIPE FAIL COMMAND: ran"
	return $PLUGRET_Normal
}

secopen() {
	say "SECURITY CHECK: secopen"
	return $PLUGRET_Normal
}

secadmincmd() {
	say "SECURITY CHECK: secadmincmd"
	return $PLUGRET_Normal
}

secauthz() {
	say "SECURITY CHECK: secauthz"
	return $PLUGRET_Normal
}

secauthall() {
	say "SECURITY CHECK: secauthall"
	return $PLUGRET_Normal
}

secelevated() {
	say "SECURITY CHECK: secelevated"
	return $PLUGRET_Normal
}

secimmediate() {
	say "SECURITY CHECK: secimmediate"
	return $PLUGRET_Normal
}

sechiddenok() {
	say "SECURITY CHECK: sechiddenok"
	return $PLUGRET_Normal
}

sechiddendenied() {
	say "SECURITY CHECK: sechiddendenied"
	return $PLUGRET_Normal
}

secadminonly() {
	say "SECURITY CHECK: secadminonly"
	return $PLUGRET_Normal
}

secusersonly() {
	say "SECURITY CHECK: secusersonly"
	return $PLUGRET_Normal
}

command=$1
shift

case "$command" in
	configure)
		default_config
		;;
	init)
		exit 0
		;;
	sendmsg|configtest|subscribe|prompts|memoryseed|memorycheck|memorythreadcheck|memorydelete|memorythreaddelete|identity|parameteraddtask|utilities|admincheck|elevatecheck|pipelineok|pipelinefail|spawnjob|pipeaddcmd|pipefinalcmd|pipefailcmd|secopen|secadmincmd|secauthz|secauthall|secelevated|secimmediate|sechiddenok|sechiddendenied|secadminonly|secusersonly)
		"$command" "$@"
		;;
	*)
		exit $PLUGRET_NotFound
		;;
esac
