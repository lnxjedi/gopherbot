output "gopherbot_service_account_email" {
  value = google_service_account.gopherbot.email
}

output "gopherbot_service_account_unique_id" {
  value       = google_service_account.gopherbot.unique_id
  description = "Use this Client ID for Domain-Wide Delegation in the Google Workspace Admin Console."
}

output "chat_topic_id" {
  value = google_pubsub_topic.chat_topic.id
}

output "chat_subscription_id" {
  value = google_pubsub_subscription.chat_subscription.id
}

output "service_account_key" {
  value     = google_service_account_key.gopherbot_key.private_key
  sensitive = true
}

output "instructions" {
  value = "Save the 'service_account_key' output to a file named 'gopherbot-key.json'. You can use 'terraform output -raw service_account_key | base64 --decode > gopherbot-key.json'"
}
