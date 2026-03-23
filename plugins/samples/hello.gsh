#!/bin/sh

default_config() {
cat <<'EOF'
---
Commands:
- Regex: (?i:hello gsh)
  Command: hello
EOF
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
	hello)
		say "Hello, Gopherbot shell World!"
		;;
	*)
		exit $PLUGRET_NotFound
		;;
esac
