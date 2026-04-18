#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

load_gcloud_env "${1:-}"

if project_exists; then
  echo "Project ${PROJECT_ID} already exists"
else
  echo "Creating project ${PROJECT_ID} (${ROBOT_NAME})"
  gcloud projects create "${PROJECT_ID}" --name="${ROBOT_NAME}"
fi

set_active_project

echo
echo "Project is ready to inspect in the web UI:"
echo "  ${PROJECT_ID}"
echo "Next step: verify billing is attached before enabling APIs or creating resources."
