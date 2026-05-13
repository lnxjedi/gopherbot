#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

# Load environment to get REGION, PROJECT_ID, and QUEUE_WEBHOOK_UUID
load_gcloud_env "${1:-}"

if [[ -z "${QUEUE_WEBHOOK_UUID:-}" ]]; then
  echo "Error: QUEUE_WEBHOOK_UUID is not set in the environment." >&2
  exit 1
fi

echo "https://${REGION}-${PROJECT_ID}.cloudfunctions.net/job-queue-${QUEUE_WEBHOOK_UUID}"