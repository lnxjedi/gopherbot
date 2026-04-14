#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

load_gcloud_env "${1:-}"
set_active_project

SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_ID}@${PROJECT_ID}.iam.gserviceaccount.com"

if [[ -e "${SERVICE_ACCOUNT_KEY_JSON}" ]]; then
  echo "Refusing to overwrite existing key file: ${SERVICE_ACCOUNT_KEY_JSON}" >&2
  echo "Move it aside or edit SERVICE_ACCOUNT_KEY_JSON in gcloud.env before retrying." >&2
  exit 1
fi

mkdir -p "$(dirname "${SERVICE_ACCOUNT_KEY_JSON}")"

echo "Creating key for ${SERVICE_ACCOUNT_EMAIL}"
gcloud iam service-accounts keys create "${SERVICE_ACCOUNT_KEY_JSON}" \
  --iam-account="${SERVICE_ACCOUNT_EMAIL}"

echo
echo "Created ${SERVICE_ACCOUNT_KEY_JSON}"
echo "Next step: encrypt it into your robot's custom/ directory, for example:"
echo "  gopherbot encrypt -f ${SERVICE_ACCOUNT_KEY_JSON} > /path/to/robot/custom/gopherbot-key.json.enc"
echo "Then remove the plaintext JSON when you are done."
