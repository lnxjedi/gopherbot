#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

load_gcloud_env "${1:-}"

echo "Adding project ${PROJECT_ID}"
gcloud create project "${PROJECT_ID}"
echo "Setting active project to ${PROJECT_ID}"
gcloud config set project "${PROJECT_ID}"

echo "Enabling required project services for Gopherbot Google Cloud setup"
gcloud services enable \
  firestore.googleapis.com \
  chat.googleapis.com \
  pubsub.googleapis.com \
  workspaceevents.googleapis.com

echo "Project services enabled for ${PROJECT_ID}"
