#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

load_gcloud_env "${1:-}"
set_active_project

SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
JOB_WEBHOOK_TOPIC="job-triggers"
JOB_WEBHOOK_SUB="job-triggers-pull"

create_job_webhook_topic() {
  gcloud pubsub topics create "${JOB_WEBHOOK_TOPIC}"
}

create_job_webhook_subscription() {
  gcloud pubsub subscriptions create "${JOB_WEBHOOK_SUB}" \
    --topic="${JOB_WEBHOOK_TOPIC}" \
    --ack-deadline=60
}

grant_subscriber_permissions() {
  echo "Granting roles/pubsub.subscriber to ${SERVICE_ACCOUNT_EMAIL} on ${JOB_WEBHOOK_SUB}"
  gcloud pubsub subscriptions add-iam-policy-binding "${JOB_WEBHOOK_SUB}" \
    --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
    --role="roles/pubsub.subscriber" >/dev/null
}

echo "Ensuring Pub/Sub topic ${JOB_WEBHOOK_TOPIC} exists"
if gcloud pubsub topics describe "${JOB_WEBHOOK_TOPIC}" >/dev/null 2>&1; then
  echo "Pub/Sub topic ${JOB_WEBHOOK_TOPIC} already exists"
else
  retry_command 8 10 create_job_webhook_topic
fi

echo "Ensuring Pub/Sub subscription ${JOB_WEBHOOK_SUB} exists"
if gcloud pubsub subscriptions describe "${JOB_WEBHOOK_SUB}" >/dev/null 2>&1; then
  echo "Pub/Sub subscription ${JOB_WEBHOOK_SUB} already exists"
else
  retry_command 8 10 create_job_webhook_subscription
fi

echo "Checking IAM permissions for the robot"
# Check if the service account already has the subscriber role on this specific subscription
CURRENT_IAM=$(gcloud pubsub subscriptions get-iam-policy "${JOB_WEBHOOK_SUB}" --format="json")
if echo "${CURRENT_IAM}" | grep -q "${SERVICE_ACCOUNT_EMAIL}.*roles/pubsub.subscriber"; then
  echo "Robot already has subscriber permissions on ${JOB_WEBHOOK_SUB}"
else
  retry_command 8 10 grant_subscriber_permissions
fi

echo
echo "Project resources are ready:"
echo "  Job webhook topic: ${JOB_WEBHOOK_TOPIC}"
echo "  Job webhook subscription: projects/${PROJECT_ID}/subscriptions/${JOB_WEBHOOK_SUB}"
echo "  Service Account: ${SERVICE_ACCOUNT_EMAIL} (Authorized Subscriber)"