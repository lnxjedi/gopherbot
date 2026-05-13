#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${WEBHOOK_URL:-}" ]]; then
  echo "Error: WEBHOOK_URL must be set." >&2
  exit 1
fi

if [[ -z "${JOB_UUID:-}" ]]; then
  echo "Error: JOB_UUID must be set." >&2
  exit 1
fi

# Use printf %q to shell-escape each argument individually.
# This ensures that "two three" becomes something like 'two three' or two\ three
# so that the receiver knows it is a single argument.
QUOTED_ARGS=$(printf "%q " "$@")

# Remove the trailing space from QUOTED_ARGS
PAYLOAD="${JOB_UUID} ${QUOTED_ARGS% }"

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${WEBHOOK_URL}" \
  -H "Content-Type: text/plain" \
  --data-binary "${PAYLOAD}")

if [[ "${HTTP_STATUS}" -eq 202 ]]; then
  echo "Job queued successfully."
else
  echo "Error: Failed to queue job (HTTP ${HTTP_STATUS})" >&2
  exit 1
fi