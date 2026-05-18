#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "ERROR: setuid-nobody.sh must be run as root" >&2
  exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Get the GID of the nobody user (works on both systems with 'nobody' or 'nogroup')
nobody_gid=$(id -g nobody) || {
  echo "ERROR: unable to resolve nobody user GID" >&2
  exit 1
}

targets=(
  "gopherbot"
  "gopherbot-integration"
)

for target in "${targets[@]}"; do
  target_path="${script_dir}/${target}"

  if [[ ! -e "${target_path}" ]]; then
    echo "ERROR: missing binary ${target_path}" >&2
    exit 1
  fi

  if [[ ! -f "${target_path}" ]]; then
    echo "ERROR: expected regular file at ${target_path}" >&2
    exit 1
  fi

  chown nobody:"${nobody_gid}" "${target_path}"
  chmod u+s,g+s "${target_path}"
  echo "Updated ${target_path}: owner nobody:${nobody_gid}, setuid/setgid enabled"
done
