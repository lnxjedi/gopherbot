#!/bin/sh

set -eu

robot=${GOPHERBOT:-./gopherbot}
fixture=test/wireguard/fixture.yaml
poisoned_fixture=test/wireguard/poisoned-state-fixture.yaml
plugin=plugins/wireguard.lua
valid_key='xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg='
invalid_message="Invalid WireGuard public key; expected the 44-character base64 value produced by 'wg pubkey'."

fail() {
  printf '%s\n' "wireguard plugin check failed: $1" >&2
  exit 1
}

run_add_device() {
  "$robot" script -json -no-interactive -fixture "$fixture" "$plugin" -- add-device test-device "$1"
}

expect_rejected() {
  label=$1
  key=$2
  output=$(run_add_device "$key") || fail "$label invocation"
  case "$output" in
    *"$invalid_message"*) ;;
    *) fail "$label was not rejected with the expected message" ;;
  esac
  case "$output" in
    *'"method": "UpdateDatum"'*) fail "$label updated brain state" ;;
  esac
}

valid_output=$(run_add_device "$valid_key") || fail "valid key invocation"
case "$valid_output" in
  *"Dry-run: VPN device 'test-device' would be added"*) ;;
  *) fail "valid key was not accepted" ;;
esac
case "$valid_output" in
  *'"method": "UpdateDatum"'*) fail "development dry-run updated brain state" ;;
esac

expect_rejected "short key" "garbage"
expect_rejected "long key" "${valid_key}A"
expect_rejected "invalid alphabet" '!TIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg='
expect_rejected "missing padding" 'xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8DgA'
expect_rejected "non-canonical padding bits" 'xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dh='

poisoned_output=$("$robot" script -json -no-interactive -fixture "$poisoned_fixture" "$plugin" -- _init) || fail "poisoned state invocation"
case "$poisoned_output" in
  *"stored state validation failed: user=alice device=broken-device field=public key"*) ;;
  *) fail "poisoned stored key was not identified" ;;
esac
case "$poisoned_output" in
  *"sudo write start"*|*"systemctl restart"*) fail "poisoned state reached a host action" ;;
esac

printf '%s\n' "wireguard plugin checks passed"

