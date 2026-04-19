#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

load_gcloud_env "${1:-}"
set_active_project

SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
TOPIC_NAME="projects/${PROJECT_ID}/topics/${TOPIC_ID}"

create_firestore_database() {
  gcloud firestore databases create \
    --database="${FIRESTORE_DATABASE_ID}" \
    --location="${REGION}" \
    --type=firestore-native
}

create_pubsub_topic() {
  gcloud pubsub topics create "${TOPIC_ID}"
}

grant_chat_publish_permission() {
  gcloud pubsub topics add-iam-policy-binding "${TOPIC_ID}" \
    --member="serviceAccount:chat-api-push@system.gserviceaccount.com" \
    --role="roles/pubsub.publisher" >/dev/null
}

create_pubsub_subscription() {
  gcloud pubsub subscriptions create "${SUBSCRIPTION_ID}" \
    --topic="${TOPIC_ID}" \
    --expiration-period=never
}

create_service_account() {
  gcloud iam service-accounts create "${SERVICE_ACCOUNT_ID}" \
    --display-name="${ROBOT_NAME} Robot Service Account"
}

grant_project_role() {
  local role="$1"
  gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
    --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
    --role="${role}" >/dev/null
}

grant_subscription_role() {
  local role="$1"
  gcloud pubsub subscriptions add-iam-policy-binding "${SUBSCRIPTION_ID}" \
    --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
    --role="${role}" >/dev/null
}

echo "Ensuring Firestore database ${FIRESTORE_DATABASE_ID} exists"
if gcloud firestore databases describe --database="${FIRESTORE_DATABASE_ID}" >/dev/null 2>&1; then
  echo "Firestore database ${FIRESTORE_DATABASE_ID} already exists"
else
  retry_command 5 10 create_firestore_database
fi

echo "Ensuring Pub/Sub topic ${TOPIC_ID} exists"
if gcloud pubsub topics describe "${TOPIC_ID}" >/dev/null 2>&1; then
  echo "Pub/Sub topic ${TOPIC_ID} already exists"
else
  retry_command 8 10 create_pubsub_topic
fi

echo "Ensuring Chat can publish to ${TOPIC_ID}"
retry_command 5 5 grant_chat_publish_permission

echo "Ensuring Pub/Sub subscription ${SUBSCRIPTION_ID} exists"
if gcloud pubsub subscriptions describe "${SUBSCRIPTION_ID}" >/dev/null 2>&1; then
  echo "Pub/Sub subscription ${SUBSCRIPTION_ID} already exists"
else
  retry_command 8 10 create_pubsub_subscription
fi

echo "Ensuring service account ${SERVICE_ACCOUNT_ID} exists"
if gcloud iam service-accounts describe "${SERVICE_ACCOUNT_EMAIL}" >/dev/null 2>&1; then
  echo "Service account ${SERVICE_ACCOUNT_EMAIL} already exists"
else
  retry_command 5 5 create_service_account
fi

echo "Granting Firestore project role to ${SERVICE_ACCOUNT_EMAIL}"
retry_command 5 5 grant_project_role "roles/datastore.user"

echo "Granting Pub/Sub subscription roles to ${SERVICE_ACCOUNT_EMAIL}"
retry_command 5 5 grant_subscription_role "roles/pubsub.subscriber"
retry_command 5 5 grant_subscription_role "roles/pubsub.viewer"

echo
echo "Project resources are ready:"
echo "  Firestore database: ${FIRESTORE_DATABASE_ID}"
echo "  Firestore collection: ${FIRESTORE_COLLECTION}"
echo "  Chat topic: ${TOPIC_NAME}"
echo "  Chat subscription: projects/${PROJECT_ID}/subscriptions/${SUBSCRIPTION_ID}"
echo "  Service account: ${SERVICE_ACCOUNT_EMAIL}"
