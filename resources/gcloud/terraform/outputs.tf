output "gopherbot_service_account_email" {
  value = google_service_account.gopherbot.email
}

output "gopherbot_service_account_unique_id" {
  value       = google_service_account.gopherbot.unique_id
  description = "Service account unique ID for reference. Ambient Google Chat access uses app authentication with administrator approval, not Domain-Wide Delegation."
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
  value = <<-EOT
  Save the 'service_account_key' output to a file named 'gopherbot-key.json':
    terraform output -raw service_account_key | base64 --decode > gopherbot-key.json

  From your robot's custom/ directory, encrypt it into the default runtime filename:
    gopherbot encrypt -f path/to/gopherbot-key.json > gopherbot-key.json.enc

  The same encrypted file can be used by both the Firestore brain and the Google Chat connector.
  EOT
}
