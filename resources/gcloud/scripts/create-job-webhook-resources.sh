#!/usr/bin/env bash

set -euo pipefail

# Get the directory where the script resides
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

load_gcloud_env "${1:-}"
set_active_project

# Generate the function name with a UUID
FUNCTION_NAME="job-queue-${QUEUE_WEBHOOK_UUID}"
JOB_WEBHOOK_TOPIC="job-triggers"

# Identity for the Cloud Function to use when publishing
PUBLISHER_SA_ID="job-webhook-publisher"
PUBLISHER_SA_EMAIL="${PUBLISHER_SA_ID}@${PROJECT_ID}.iam.gserviceaccount.com"

# --- Infrastructure Setup ---

echo "Ensuring Publisher Service Account exists..."
if ! gcloud iam service-accounts describe "${PUBLISHER_SA_EMAIL}" >/dev/null 2>&1; then
  gcloud iam service-accounts create "${PUBLISHER_SA_ID}" \
    --display-name="Webhook to PubSub Publisher"  
  # CRITICAL: IAM propagation is eventually consistent.
  # We wait a few seconds to ensure the identity exists globally.
  echo "Waiting for IAM propagation..."
  sleep 10
fi

echo "Ensuring Publisher SA has permission to publish to ${JOB_WEBHOOK_TOPIC}"
# We'll use the retry_command function you likely have in lib.sh 
# to handle any lingering propagation issues.
retry_command 5 5 gcloud pubsub topics add-iam-policy-binding "${JOB_WEBHOOK_TOPIC}" \
  --member="serviceAccount:${PUBLISHER_SA_EMAIL}" \
  --role="roles/pubsub.publisher"

# --- Deployment ---

echo "Deploying Cloud Function: ${FUNCTION_NAME}"
# Pointing source to the 'webhook' subdirectory relative to SCRIPT_DIR
gcloud functions deploy "${FUNCTION_NAME}" \
  --gen2 \
  --runtime=nodejs24 \
  --region="${REGION}" \
  --source="${SCRIPT_DIR}/webhook" \
  --entry-point=webhookIngest \
  --trigger-http \
  --allow-unauthenticated \
  --service-account="${PUBLISHER_SA_EMAIL}" \
  --set-env-vars PROJECT_ID="${PROJECT_ID}",JOB_WEBHOOK_TOPIC="${JOB_WEBHOOK_TOPIC}"

echo
echo "Webhook Provisioned Successfully."
echo "----------------------------------------------------------------"
echo "Target URL: https://${REGION}-${PROJECT_ID}.cloudfunctions.net/${FUNCTION_NAME}"
echo "----------------------------------------------------------------"