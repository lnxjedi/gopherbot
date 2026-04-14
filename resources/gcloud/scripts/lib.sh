#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GCLOUD_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEFAULT_ENV_FILE="${GCLOUD_DIR}/gcloud.env"

load_gcloud_env() {
  local env_file="${1:-${DEFAULT_ENV_FILE}}"
  if [[ ! -f "${env_file}" ]]; then
    echo "Missing env file: ${env_file}" >&2
    echo "Copy ${GCLOUD_DIR}/gcloud.env.example to ${GCLOUD_DIR}/gcloud.env, edit it, then retry." >&2
    exit 1
  fi

  # shellcheck disable=SC1090
  source "${env_file}"

  : "${PROJECT_ID:?PROJECT_ID must be set in ${env_file}}"
  : "${REGION:?REGION must be set in ${env_file}}"
  : "${TOPIC_ID:?TOPIC_ID must be set in ${env_file}}"
  : "${SUBSCRIPTION_ID:?SUBSCRIPTION_ID must be set in ${env_file}}"
  : "${SERVICE_ACCOUNT_ID:?SERVICE_ACCOUNT_ID must be set in ${env_file}}"
  : "${SERVICE_ACCOUNT_KEY_JSON:?SERVICE_ACCOUNT_KEY_JSON must be set in ${env_file}}"
  : "${FIRESTORE_DATABASE_ID:?FIRESTORE_DATABASE_ID must be set in ${env_file}}"
  : "${FIRESTORE_COLLECTION:?FIRESTORE_COLLECTION must be set in ${env_file}}"
  : "${ROBOT_NAME:?ROBOT_NAME must be set in ${env_file}}"
}

set_active_project() {
  echo "Setting active project to ${PROJECT_ID}"
  gcloud config set project "${PROJECT_ID}" >/dev/null
}
