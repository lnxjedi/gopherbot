#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

load_gcloud_env "${1:-}"
set_active_project

SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_ID}@${PROJECT_ID}.iam.gserviceaccount.com"

echo "Ensuring Firestore database ${FIRESTORE_DATABASE_ID} exists"
if gcloud firestore databases describe --database="${FIRESTORE_DATABASE_ID}" >/dev/null 2>&1; then
  echo "Firestore database ${FIRESTORE_DATABASE_ID} already exists"
else
  gcloud firestore databases create \
    --database="${FIRESTORE_DATABASE_ID}" \
    --location="${REGION}" \
    --type=firestore-native
fi

echo "Ensuring Pub/Sub topic ${TOPIC_ID} exists"
if gcloud pubsub topics describe "${TOPIC_ID}" >/dev/null 2>&1; then
  echo "Pub/Sub topic ${TOPIC_ID} already exists"
else
  if [[ -n "${PUBSUB_ALLOWED_REGIONS:-}" ]]; then
    gcloud pubsub topics create "${TOPIC_ID}" \
      --message-storage-policy-allowed-regions="${PUBSUB_ALLOWED_REGIONS}"
  else
    gcloud pubsub topics create "${TOPIC_ID}"
  fi
fi

echo "Ensuring Chat can publish to ${TOPIC_ID}"
gcloud pubsub topics add-iam-policy-binding "${TOPIC_ID}" \
  --member="serviceAccount:chat-api-push@system.gserviceaccount.com" \
  --role="roles/pubsub.publisher" >/dev/null

echo "Ensuring Pub/Sub subscription ${SUBSCRIPTION_ID} exists"
if gcloud pubsub subscriptions describe "${SUBSCRIPTION_ID}" >/dev/null 2>&1; then
  echo "Pub/Sub subscription ${SUBSCRIPTION_ID} already exists"
else
  gcloud pubsub subscriptions create "${SUBSCRIPTION_ID}" \
    --topic="${TOPIC_ID}" \
    --expiration-period=never
fi

echo "Ensuring service account ${SERVICE_ACCOUNT_ID} exists"
if gcloud iam service-accounts describe "${SERVICE_ACCOUNT_EMAIL}" >/dev/null 2>&1; then
  echo "Service account ${SERVICE_ACCOUNT_EMAIL} already exists"
else
  gcloud iam service-accounts create "${SERVICE_ACCOUNT_ID}" \
    --display-name="${ROBOT_NAME} Robot Service Account"
fi

echo "Granting project roles to ${SERVICE_ACCOUNT_EMAIL}"
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/datastore.user" >/dev/null
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/pubsub.subscriber" >/dev/null
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/pubsub.viewer" >/dev/null

echo
echo "Project resources are ready:"
echo "  Firestore database: ${FIRESTORE_DATABASE_ID}"
echo "  Firestore collection: ${FIRESTORE_COLLECTION}"
echo "  Chat topic: projects/${PROJECT_ID}/topics/${TOPIC_ID}"
echo "  Chat subscription: projects/${PROJECT_ID}/subscriptions/${SUBSCRIPTION_ID}"
echo "  Service account: ${SERVICE_ACCOUNT_EMAIL}"
