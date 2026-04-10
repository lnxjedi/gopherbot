terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Enable required APIs
resource "google_project_service" "firestore" {
  service            = "firestore.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "chat" {
  service            = "chat.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "pubsub" {
  service            = "pubsub.googleapis.com"
  disable_on_destroy = false
}

# Firestore Database (Native Mode)
resource "google_firestore_database" "database" {
  name                    = "(default)"
  location_id             = var.region
  type                    = "FIRESTORE_NATIVE"
  delete_protection_state = "DELETE_PROTECTION_DISABLED"

  depends_on = [google_project_service.firestore]
}

# Pub/Sub Topic for Google Chat
resource "google_pubsub_topic" "chat_topic" {
  name = "gopherbot-chat"

  depends_on = [google_project_service.pubsub]
}

# Pub/Sub Subscription for Gopherbot to pull messages
resource "google_pubsub_subscription" "chat_subscription" {
  name  = "gopherbot-chat-sub"
  topic = google_pubsub_topic.chat_topic.name

  # Never expire the subscription
  expiration_policy {
    ttl = ""
  }
}

# Service Account for the Robot
resource "google_service_account" "gopherbot" {
  account_id   = "gopherbot-robot"
  display_name = "Gopherbot Robot Service Account"
}

# IAM Roles for the Service Account
resource "google_project_iam_member" "firestore_owner" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.gopherbot.email}"
}

resource "google_project_iam_member" "pubsub_subscriber" {
  project = var.project_id
  role    = "roles/pubsub.subscriber"
  member  = "serviceAccount:${google_service_account.gopherbot.email}"
}

# Allow Google Chat to publish to the topic
resource "google_pubsub_topic_iam_member" "chat_publisher" {
  topic  = google_pubsub_topic.chat_topic.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:chat-api-push@system.gserviceaccount.com"
}

# Service Account Key (for the credentials file)
resource "google_service_account_key" "gopherbot_key" {
  service_account_id = google_service_account.gopherbot.name
}
